package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/task"
	"errors"

	"gorm.io/gorm"
)

var (
	ErrCourseDetailNotFound      = errors.New("course detail not found")
	ErrCourseEvaluationNotFound  = errors.New("course evaluation not found")
	ErrCourseEvaluationConflict  = errors.New("course evaluation conflict")
	ErrCourseEvaluationForbidden = errors.New("course evaluation forbidden")
)

type CourseService struct {
	courseRepo repo.CourseRepository
	socialRepo repo.SocialRepository
}

func NewCourseService(cr repo.CourseRepository, sr repo.SocialRepository) *CourseService {
	return &CourseService{courseRepo: cr, socialRepo: sr}
}

func (s *CourseService) ListCourses(query repo.CourseListQuery) ([]repo.CourseListItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.Sort == "" {
		query.Sort = "avg_score"
	}
	return s.courseRepo.FindCourses(query)
}

func (s *CourseService) GetCourseDetail(id int64) (*repo.CourseDetail, error) {
	detail, err := s.courseRepo.FindCourseDetail(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCourseDetailNotFound
	}
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (s *CourseService) ListCourseRankings(query repo.CourseRankingQuery) ([]repo.CourseRankingItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.RankType == "" {
		query.RankType = "avg_score"
	}
	if query.RankType == "hot_score" {
		query.RankType = "hot"
	}
	if query.Period == "" {
		query.Period = "all"
	}

	cacheKey := task.CourseRankingCacheKey(query.RankType, query.Period)
	ids, scores, total, err := task.ReadRankingIDs(cacheKey, query.Page, query.Size, query.IsIncreased)
	if err == nil && total > 0 {
		items, err := s.courseRepo.FindCourseRankingItemsByIDs(ids)
		if err == nil {
			itemMap := make(map[int64]repo.CourseRankingItem, len(items))
			for _, item := range items {
				itemMap[item.ID] = item
			}
			ordered := make([]repo.CourseRankingItem, 0, len(ids))
			startRank := int64((query.Page-1)*query.Size + 1)
			for i, id := range ids {
				item, ok := itemMap[id]
				if !ok {
					continue
				}
				item.Score = scores[i]
				item.Rank = startRank + int64(i)
				ordered = append(ordered, item)
			}
			return ordered, total, nil
		}
	}

	return s.courseRepo.FindCourseRankings(query)
}

func (s *CourseService) ListCourseEvaluations(query repo.CourseEvaluationQuery, userID int64) ([]repo.CourseEvaluationItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.Sort == "" {
		query.Sort = "created_at"
	}

	exists, err := s.courseRepo.CourseExists(query.CourseID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, ErrCourseDetailNotFound
	}

	items, total, err := s.courseRepo.ListCourseEvaluations(query)
	if err != nil {
		return nil, 0, err
	}

	for i := range items {
		if userID > 0 && s.socialRepo != nil {
			liked, err := s.socialRepo.HasLike(userID, model.LikeTargetTypeCourseEvaluation, items[i].ID)
			if err != nil {
				return nil, 0, err
			}
			items[i].IsLiked = liked
		}
		if items[i].IsAnonymous {
			items[i].AuthorID = 0
			items[i].AuthorName = "匿名用户"
			items[i].AuthorAvatarURL = ""
		}
	}

	return items, total, nil
}

func (s *CourseService) CreateCourseEvaluation(userID, courseID int64, input model.CourseEvaluations) (*model.CourseEvaluations, error) {
	exists, err := s.courseRepo.CourseExists(courseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCourseDetailNotFound
	}

	input.UserID = userID
	input.CourseID = courseID
	input.Status = model.ResourceStatusApproved
	if err := s.courseRepo.CreateCourseEvaluation(&input); err != nil {
		if isDuplicateKeyErr(err) {
			return nil, ErrCourseEvaluationConflict
		}
		return nil, err
	}
	if err := s.courseRepo.RecalculateCourseStats(courseID); err != nil {
		return nil, err
	}
	return &input, nil
}

func (s *CourseService) UpdateCourseEvaluation(userID int64, userRole string, evaluationID int64, input model.CourseEvaluations) (*model.CourseEvaluations, error) {
	evaluation, err := s.courseRepo.GetCourseEvaluationByID(evaluationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCourseEvaluationNotFound
	}
	if err != nil {
		return nil, err
	}

	if evaluation.UserID != userID && !isPrivilegedRole(userRole) {
		return nil, ErrCourseEvaluationForbidden
	}

	evaluation.WorkloadScore = input.WorkloadScore
	evaluation.GainScore = input.GainScore
	evaluation.DifficultyScore = input.DifficultyScore
	evaluation.Comment = input.Comment
	evaluation.IsAnonymous = input.IsAnonymous
	evaluation.Status = model.ResourceStatusApproved

	if err := s.courseRepo.UpdateCourseEvaluation(evaluation); err != nil {
		if isDuplicateKeyErr(err) {
			return nil, ErrCourseEvaluationConflict
		}
		return nil, err
	}
	if err := s.courseRepo.RecalculateCourseStats(evaluation.CourseID); err != nil {
		return nil, err
	}

	return evaluation, nil
}

func (s *CourseService) DeleteCourseEvaluation(userID int64, userRole string, evaluationID int64) error {
	evaluation, err := s.courseRepo.GetCourseEvaluationByID(evaluationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrCourseEvaluationNotFound
	}
	if err != nil {
		return err
	}

	if evaluation.UserID != userID && !isPrivilegedRole(userRole) {
		return ErrCourseEvaluationForbidden
	}

	if err := s.courseRepo.DeleteCourseEvaluation(evaluationID); err != nil {
		return err
	}

	return s.courseRepo.RecalculateCourseStats(evaluation.CourseID)
}

func (s *CourseService) ListMyCourseEvaluations(userID int64, page, size int) ([]repo.MyCourseEvaluationItem, int64, error) {
	fillPagination(&page, &size)
	return s.courseRepo.ListMyCourseEvaluations(userID, page, size)
}

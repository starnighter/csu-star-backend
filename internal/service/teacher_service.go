package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/task"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrTeacherNotFound            = errors.New("teacher not found")
	ErrCourseNotFound             = errors.New("course not found")
	ErrTeacherCourseMismatch      = errors.New("teacher course mismatch")
	ErrTeacherEvaluationNotFound  = errors.New("teacher evaluation not found")
	ErrTeacherEvaluationConflict  = errors.New("teacher evaluation conflict")
	ErrTeacherEvaluationForbidden = errors.New("teacher evaluation forbidden")
)

type TeacherService struct {
	teacherRepo repo.TeacherRepository
	socialRepo  repo.SocialRepository
}

func NewTeacherService(tr repo.TeacherRepository, sr repo.SocialRepository) *TeacherService {
	return &TeacherService{teacherRepo: tr, socialRepo: sr}
}

func (s *TeacherService) ListTeachers(query repo.TeacherListQuery) ([]repo.TeacherListItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.Sort == "" {
		query.Sort = "avg_score"
	}
	return s.teacherRepo.FindTeachers(query)
}

func (s *TeacherService) GetTeacherDetail(id int64) (*repo.TeacherDetail, error) {
	detail, err := s.teacherRepo.FindTeacherDetail(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTeacherNotFound
	}
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (s *TeacherService) ListTeacherRankings(query repo.TeacherRankingQuery) ([]repo.TeacherRankingItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.RankType == "" {
		query.RankType = "avg_score"
	}
	if query.Period == "" {
		query.Period = "all"
	}

	cacheKey := task.TeacherRankingCacheKey(query.RankType, query.Period, query.DepartmentID)
	ids, scores, total, err := task.ReadRankingIDs(cacheKey, query.Page, query.Size, query.IsIncreased)
	if err == nil && total > 0 {
		items, err := s.teacherRepo.FindTeacherRankingItemsByIDs(ids)
		if err == nil {
			itemMap := make(map[int64]repo.TeacherRankingItem, len(items))
			for _, item := range items {
				itemMap[item.ID] = item
			}
			ordered := make([]repo.TeacherRankingItem, 0, len(ids))
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

	return s.teacherRepo.FindTeacherRankings(query)
}

func (s *TeacherService) ListTeacherEvaluations(query repo.TeacherEvaluationQuery, userID int64) ([]repo.TeacherEvaluationItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.Sort == "" {
		query.Sort = "created_at"
	}

	exists, err := s.teacherRepo.TeacherExists(query.TeacherID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, ErrTeacherNotFound
	}

	items, total, err := s.teacherRepo.ListTeacherEvaluations(query)
	if err != nil {
		return nil, 0, err
	}

	for i := range items {
		if userID > 0 && s.socialRepo != nil {
			liked, err := s.socialRepo.HasLike(userID, model.LikeTargetTypeTeacherEvaluation, items[i].ID)
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

func (s *TeacherService) CreateTeacherEvaluation(userID, teacherID int64, input model.TeacherEvaluations) (*model.TeacherEvaluations, error) {
	if err := s.validateTeacherEvaluationRefs(teacherID, input.CourseID); err != nil {
		return nil, err
	}

	input.UserID = userID
	input.TeacherID = teacherID
	input.Status = model.ResourceStatusApproved
	if err := s.teacherRepo.CreateTeacherEvaluation(&input); err != nil {
		if isDuplicateKeyErr(err) {
			return nil, ErrTeacherEvaluationConflict
		}
		return nil, err
	}
	if err := s.teacherRepo.RecalculateTeacherStats(teacherID); err != nil {
		return nil, err
	}
	return &input, nil
}

func (s *TeacherService) UpdateTeacherEvaluation(userID int64, userRole string, evaluationID int64, input model.TeacherEvaluations) (*model.TeacherEvaluations, error) {
	evaluation, err := s.teacherRepo.GetTeacherEvaluationByID(evaluationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTeacherEvaluationNotFound
	}
	if err != nil {
		return nil, err
	}

	if evaluation.UserID != userID && !isPrivilegedRole(userRole) {
		return nil, ErrTeacherEvaluationForbidden
	}

	if err := s.validateTeacherEvaluationRefs(evaluation.TeacherID, input.CourseID); err != nil {
		return nil, err
	}

	evaluation.CourseID = input.CourseID
	evaluation.TeachingScore = input.TeachingScore
	evaluation.GradingScore = input.GradingScore
	evaluation.AttendanceScore = input.AttendanceScore
	evaluation.Comment = input.Comment
	evaluation.IsAnonymous = input.IsAnonymous
	evaluation.Status = model.ResourceStatusApproved

	if err := s.teacherRepo.UpdateTeacherEvaluation(evaluation); err != nil {
		if isDuplicateKeyErr(err) {
			return nil, ErrTeacherEvaluationConflict
		}
		return nil, err
	}
	if err := s.teacherRepo.RecalculateTeacherStats(evaluation.TeacherID); err != nil {
		return nil, err
	}

	return evaluation, nil
}

func (s *TeacherService) DeleteTeacherEvaluation(userID int64, userRole string, evaluationID int64) error {
	evaluation, err := s.teacherRepo.GetTeacherEvaluationByID(evaluationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrTeacherEvaluationNotFound
	}
	if err != nil {
		return err
	}

	if evaluation.UserID != userID && !isPrivilegedRole(userRole) {
		return ErrTeacherEvaluationForbidden
	}

	if err := s.teacherRepo.DeleteTeacherEvaluation(evaluationID); err != nil {
		return err
	}

	return s.teacherRepo.RecalculateTeacherStats(evaluation.TeacherID)
}

func (s *TeacherService) ListMyTeacherEvaluations(userID int64, page, size int) ([]repo.MyTeacherEvaluationItem, int64, error) {
	fillPagination(&page, &size)
	return s.teacherRepo.ListMyTeacherEvaluations(userID, page, size)
}

func (s *TeacherService) validateTeacherEvaluationRefs(teacherID, courseID int64) error {
	teacherExists, err := s.teacherRepo.TeacherExists(teacherID)
	if err != nil {
		return err
	}
	if !teacherExists {
		return ErrTeacherNotFound
	}

	courseExists, err := s.teacherRepo.CourseExists(courseID)
	if err != nil {
		return err
	}
	if !courseExists {
		return ErrCourseNotFound
	}

	related, err := s.teacherRepo.TeacherCourseRelationExists(teacherID, courseID)
	if err != nil {
		return err
	}
	if !related {
		return ErrTeacherCourseMismatch
	}
	return nil
}

func fillPagination(page, size *int) {
	if *page <= 0 {
		*page = 1
	}
	if *size <= 0 {
		*size = 10
	}
	if *size >= 50 {
		*size = 50
	}
}

func isPrivilegedRole(role string) bool {
	return role == string(model.UserRoleAdmin) || role == string(model.UserRoleAuditor)
}

func isDuplicateKeyErr(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}

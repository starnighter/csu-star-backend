package service

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/task"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrCourseDetailNotFound           = errors.New("course detail not found")
	ErrCourseEvaluationNotFound       = errors.New("course evaluation not found")
	ErrCourseEvaluationConflict       = errors.New("course evaluation conflict")
	ErrCourseEvaluationForbidden      = errors.New("course evaluation forbidden")
	ErrCourseTeacherRelationConflict  = errors.New("course teacher relation conflict")
	ErrCourseTeacherRelationNotFound  = errors.New("course teacher relation not found")
	ErrCourseEvaluationReplyNotFound  = errors.New("course evaluation reply not found")
	ErrCourseEvaluationReplyForbidden = errors.New("course evaluation reply forbidden")
)

type CourseService struct {
	db          *gorm.DB
	courseRepo  repo.CourseRepository
	teacherRepo repo.TeacherRepository
	socialRepo  repo.SocialRepository
}

func NewCourseService(db *gorm.DB, cr repo.CourseRepository, tr repo.TeacherRepository, sr repo.SocialRepository) *CourseService {
	return &CourseService{db: db, courseRepo: cr, teacherRepo: tr, socialRepo: sr}
}

func (s *CourseService) ListCourses(query repo.CourseListQuery) ([]repo.CourseListItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.Sort == "" {
		query.Sort = "avg_score"
	}
	return s.courseRepo.FindCourses(query)
}

func (s *CourseService) ListSimpleCourses(q string) ([]repo.CourseSimpleItem, error) {
	return s.courseRepo.ListSimpleCourses(q, 20)
}

func (s *CourseService) GetCourseDetail(id int64, viewerID int64) (*repo.CourseDetail, error) {
	detail, err := s.courseRepo.FindCourseDetail(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCourseDetailNotFound
	}
	if err != nil {
		return nil, err
	}
	if viewerID > 0 {
		favorited, err := s.socialRepo.HasFavorite(viewerID, model.FavoriteTargetTypeCourse, detail.ID)
		if err != nil {
			return nil, err
		}
		detail.IsFavorited = favorited
	}
	return detail, nil
}

func (s *CourseService) GetCourseResourceCollectionDetail(query repo.CourseResourceCollectionQuery) (*repo.CourseResourceCollectionDetail, error) {
	fillPagination(&query.Page, &query.Size)
	detail, err := s.courseRepo.FindCourseResourceCollectionDetail(query)
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
	switch query.RankType {
	case "", "avg_score":
		query.RankType = "avg_score"
	case "avg_homework", "avg_gain", "avg_exam_diff", "resource_count", "favorite_count":
	default:
		query.RankType = "avg_score"
	}
	query.Period = "all"

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
			if err := s.attachCourseRankingTeachers(ordered); err != nil {
				return nil, 0, err
			}
			return ordered, total, nil
		}
	}

	items, total, err := s.courseRepo.FindCourseRankings(query)
	if err != nil {
		return nil, 0, err
	}
	if err := s.attachCourseRankingTeachers(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
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

	if err := s.attachCourseEvaluationLikeState(items, userID); err != nil {
		return nil, 0, err
	}
	s.normalizeCourseEvaluationUsers(items)

	return items, total, nil
}

func (s *CourseService) CreateCourseEvaluation(userID, courseID int64, input model.CourseEvaluations) (*model.CourseEvaluations, error) {
	if err := s.validateCourseEvaluationRefs(courseID, input.TeacherID); err != nil {
		return nil, err
	}
	if err := validateCourseEvaluationPayload(input.TeacherID, input.TeachingScore, input.GradingScore, input.AttendanceScore); err != nil {
		return nil, err
	}
	if _, err := s.courseRepo.FindCourseEvaluationByContext(userID, courseID, input.TeacherID, courseEvaluationMode(input.TeacherID)); err == nil {
		return nil, ErrCourseEvaluationConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	input.UserID = userID
	input.CourseID = courseID
	input.Status = model.ResourceStatusApproved
	input.Mode = courseEvaluationMode(input.TeacherID)
	if input.Mode == model.EvaluationModeLinked {
		input.MirrorEntityType = model.MirrorEntityTypeTeacher
	} else {
		input.MirrorEntityType = ""
	}
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&input).Error; err != nil {
			if isDuplicateKeyErr(err) {
				return ErrCourseEvaluationConflict
			}
			return err
		}
		if input.Mode != model.EvaluationModeLinked {
			if err := s.courseRepoWithTx(tx).RecalculateCourseStats(courseID); err != nil {
				return err
			}
			return rewardUserPointsTx(
				tx,
				userID,
				input.ID,
				evaluationRewardPoints,
				model.PointsTypeManual,
				"evaluation_reward",
				"评价已发布",
				"你发表评价获得了 1 积分。",
			)
		}
		teacherEval := &model.TeacherEvaluations{
			UserID:              userID,
			TeacherID:           *input.TeacherID,
			CourseID:            &courseID,
			Mode:                model.EvaluationModeLinked,
			MirrorEntityType:    model.MirrorEntityTypeCourse,
			TeachingScore:       valueOrZero(input.TeachingScore),
			GradingScore:        valueOrZero(input.GradingScore),
			AttendanceScore:     valueOrZero(input.AttendanceScore),
			HomeworkScore:       intPtr(input.WorkloadScore),
			GainScore:           intPtr(input.GainScore),
			ExamDifficultyScore: intPtr(input.DifficultyScore),
			Comment:             input.Comment,
			IsAnonymous:         input.IsAnonymous,
			Status:              model.ResourceStatusApproved,
		}
		if err := tx.Create(teacherEval).Error; err != nil {
			if isDuplicateKeyErr(err) {
				return ErrCourseEvaluationConflict
			}
			return err
		}
		input.MirrorEvaluationID = &teacherEval.ID
		teacherEval.MirrorEvaluationID = &input.ID
		if err := tx.Model(&model.CourseEvaluations{}).Where("id = ?", input.ID).Updates(map[string]interface{}{
			"mirror_evaluation_id": input.MirrorEvaluationID,
			"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.TeacherEvaluations{}).Where("id = ?", teacherEval.ID).Updates(map[string]interface{}{
			"mirror_evaluation_id": teacherEval.MirrorEvaluationID,
			"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error; err != nil {
			return err
		}
		if err := s.courseRepoWithTx(tx).RecalculateCourseStats(courseID); err != nil {
			return err
		}
		if err := s.teacherRepoWithTx(tx).RecalculateTeacherStats(*input.TeacherID); err != nil {
			return err
		}
		return rewardUserPointsTx(
			tx,
			userID,
			input.ID,
			evaluationRewardPoints,
			model.PointsTypeManual,
			"evaluation_reward",
			"评价已发布",
			"你发表评价获得了 1 积分。",
		)
	})
	if txErr != nil {
		return nil, txErr
	}
	return &input, nil
}

func (s *CourseService) CreateCourseTeacherRelation(courseID, teacherID int64) (*model.CourseTeachers, error) {
	exists, err := s.courseRepo.CourseExists(courseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCourseDetailNotFound
	}

	teacherExists, err := s.teacherRepo.TeacherExists(teacherID)
	if err != nil {
		return nil, err
	}
	if !teacherExists {
		return nil, ErrTeacherNotFound
	}

	var relation *model.CourseTeachers
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		teacherRepoTx := s.teacherRepoWithTx(tx)
		existing, err := teacherRepoTx.GetCourseTeacherRelation(courseID, teacherID)
		switch {
		case err == nil:
			if existing.Status == model.CourseTeacherRelationStatusActive {
				return ErrCourseTeacherRelationConflict
			}
			existing.Status = model.CourseTeacherRelationStatusActive
			existing.CanceledAt = nil
			if err := teacherRepoTx.UpdateCourseTeacherRelation(existing); err != nil {
				return err
			}
			if err := s.courseRepoWithTx(tx).RecalculateCourseStats(courseID); err != nil {
				return err
			}
			if err := teacherRepoTx.RecalculateTeacherStats(teacherID); err != nil {
				return err
			}
			relation = existing
			return nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		newRelation := &model.CourseTeachers{
			CourseID:  courseID,
			TeacherID: teacherID,
			Status:    model.CourseTeacherRelationStatusActive,
		}
		if err := teacherRepoTx.CreateCourseTeacherRelation(newRelation); err != nil {
			if isDuplicateKeyErr(err) {
				return ErrCourseTeacherRelationConflict
			}
			return err
		}
		if err := s.courseRepoWithTx(tx).RecalculateCourseStats(courseID); err != nil {
			return err
		}
		if err := teacherRepoTx.RecalculateTeacherStats(teacherID); err != nil {
			return err
		}
		relation = newRelation
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return relation, nil
}

func (s *CourseService) CancelCourseTeacherRelation(courseID, teacherID int64) error {
	exists, err := s.courseRepo.CourseExists(courseID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCourseDetailNotFound
	}

	teacherExists, err := s.teacherRepo.TeacherExists(teacherID)
	if err != nil {
		return err
	}
	if !teacherExists {
		return ErrTeacherNotFound
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		relation, err := s.teacherRepoWithTx(tx).GetCourseTeacherRelation(courseID, teacherID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCourseTeacherRelationNotFound
		}
		if err != nil {
			return err
		}
		if relation.Status != model.CourseTeacherRelationStatusActive {
			return ErrCourseTeacherRelationNotFound
		}
		now := time.Now()
		relation.Status = model.CourseTeacherRelationStatusCanceled
		relation.CanceledAt = &now
		if err := s.teacherRepoWithTx(tx).UpdateCourseTeacherRelation(relation); err != nil {
			return err
		}
		if err := s.courseRepoWithTx(tx).RecalculateCourseStats(courseID); err != nil {
			return err
		}
		return s.teacherRepoWithTx(tx).RecalculateTeacherStats(teacherID)
	})
}

func (s *CourseService) GetCourseEvaluationItem(id int64, viewerID int64) (*repo.CourseEvaluationItem, error) {
	item, err := s.courseRepo.GetCourseEvaluationItemByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCourseEvaluationNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.hydrateCourseEvaluationItem(item, viewerID); err != nil {
		return nil, err
	}
	return item, nil
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

	if err := s.validateCourseEvaluationRefs(evaluation.CourseID, input.TeacherID); err != nil {
		return nil, err
	}
	if err := validateCourseEvaluationPayload(input.TeacherID, input.TeachingScore, input.GradingScore, input.AttendanceScore); err != nil {
		return nil, err
	}

	oldTeacherID := evaluation.TeacherID
	oldMirrorID := evaluation.MirrorEvaluationID
	evaluation.TeacherID = input.TeacherID
	evaluation.Mode = courseEvaluationMode(input.TeacherID)
	if evaluation.Mode == model.EvaluationModeLinked {
		evaluation.MirrorEntityType = model.MirrorEntityTypeTeacher
	} else {
		evaluation.MirrorEntityType = ""
	}
	evaluation.WorkloadScore = input.WorkloadScore
	evaluation.GainScore = input.GainScore
	evaluation.DifficultyScore = input.DifficultyScore
	evaluation.TeachingScore = input.TeachingScore
	evaluation.GradingScore = input.GradingScore
	evaluation.AttendanceScore = input.AttendanceScore
	evaluation.Comment = input.Comment
	evaluation.IsAnonymous = input.IsAnonymous
	evaluation.Status = model.ResourceStatusApproved
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		switch {
		case oldMirrorID == nil && evaluation.Mode == model.EvaluationModeSingle:
			if err := tx.Model(&model.CourseEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"teacher_id":           nil,
				"mode":                 evaluation.Mode,
				"mirror_evaluation_id": nil,
				"mirror_entity_type":   nil,
				"workload_score":       evaluation.WorkloadScore,
				"gain_score":           evaluation.GainScore,
				"difficulty_score":     evaluation.DifficultyScore,
				"teaching_score":       nil,
				"grading_score":        nil,
				"attendance_score":     nil,
				"comment":              evaluation.Comment,
				"is_anonymous":         evaluation.IsAnonymous,
				"status":               evaluation.Status,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrCourseEvaluationConflict
				}
				return err
			}
		case oldMirrorID == nil && evaluation.Mode == model.EvaluationModeLinked:
			mirror := &model.TeacherEvaluations{
				UserID:              evaluation.UserID,
				TeacherID:           *evaluation.TeacherID,
				CourseID:            int64Ptr(evaluation.CourseID),
				Mode:                model.EvaluationModeLinked,
				MirrorEntityType:    model.MirrorEntityTypeCourse,
				TeachingScore:       valueOrZero(evaluation.TeachingScore),
				GradingScore:        valueOrZero(evaluation.GradingScore),
				AttendanceScore:     valueOrZero(evaluation.AttendanceScore),
				HomeworkScore:       intPtr(evaluation.WorkloadScore),
				GainScore:           intPtr(evaluation.GainScore),
				ExamDifficultyScore: intPtr(evaluation.DifficultyScore),
				Comment:             evaluation.Comment,
				IsAnonymous:         evaluation.IsAnonymous,
				Status:              evaluation.Status,
			}
			if err := tx.Model(&model.CourseEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"teacher_id":         evaluation.TeacherID,
				"mode":               evaluation.Mode,
				"mirror_entity_type": evaluation.MirrorEntityType,
				"workload_score":     evaluation.WorkloadScore,
				"gain_score":         evaluation.GainScore,
				"difficulty_score":   evaluation.DifficultyScore,
				"teaching_score":     evaluation.TeachingScore,
				"grading_score":      evaluation.GradingScore,
				"attendance_score":   evaluation.AttendanceScore,
				"comment":            evaluation.Comment,
				"is_anonymous":       evaluation.IsAnonymous,
				"status":             evaluation.Status,
				"updated_at":         gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrCourseEvaluationConflict
				}
				return err
			}
			if err := tx.Create(mirror).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrCourseEvaluationConflict
				}
				return err
			}
			evaluation.MirrorEvaluationID = &mirror.ID
			if err := tx.Model(&model.CourseEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"mirror_evaluation_id": evaluation.MirrorEvaluationID,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.TeacherEvaluations{}).Where("id = ?", mirror.ID).Updates(map[string]interface{}{
				"mirror_evaluation_id": evaluation.ID,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				return err
			}
		case oldMirrorID != nil && evaluation.Mode == model.EvaluationModeSingle:
			if err := tx.Delete(&model.TeacherEvaluations{}, *oldMirrorID).Error; err != nil {
				return err
			}
			evaluation.MirrorEvaluationID = nil
			if err := tx.Model(&model.CourseEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"teacher_id":           nil,
				"mode":                 evaluation.Mode,
				"mirror_evaluation_id": nil,
				"workload_score":       evaluation.WorkloadScore,
				"gain_score":           evaluation.GainScore,
				"difficulty_score":     evaluation.DifficultyScore,
				"teaching_score":       nil,
				"grading_score":        nil,
				"attendance_score":     nil,
				"comment":              evaluation.Comment,
				"is_anonymous":         evaluation.IsAnonymous,
				"status":               evaluation.Status,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				return err
			}
		default:
			if err := tx.Model(&model.CourseEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"teacher_id":       evaluation.TeacherID,
				"mode":             evaluation.Mode,
				"workload_score":   evaluation.WorkloadScore,
				"gain_score":       evaluation.GainScore,
				"difficulty_score": evaluation.DifficultyScore,
				"teaching_score":   evaluation.TeachingScore,
				"grading_score":    evaluation.GradingScore,
				"attendance_score": evaluation.AttendanceScore,
				"comment":          evaluation.Comment,
				"is_anonymous":     evaluation.IsAnonymous,
				"status":           evaluation.Status,
				"updated_at":       gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrCourseEvaluationConflict
				}
				return err
			}
			if err := tx.Model(&model.TeacherEvaluations{}).Where("id = ?", *oldMirrorID).Updates(map[string]interface{}{
				"course_id":            evaluation.CourseID,
				"teacher_id":           *evaluation.TeacherID,
				"mode":                 evaluation.Mode,
				"mirror_entity_type":   model.MirrorEntityTypeCourse,
				"teaching_score":       valueOrZero(evaluation.TeachingScore),
				"grading_score":        valueOrZero(evaluation.GradingScore),
				"attendance_score":     valueOrZero(evaluation.AttendanceScore),
				"workload_score":       intPtr(evaluation.WorkloadScore),
				"gain_score":           intPtr(evaluation.GainScore),
				"difficulty_score":     intPtr(evaluation.DifficultyScore),
				"comment":              evaluation.Comment,
				"is_anonymous":         evaluation.IsAnonymous,
				"status":               evaluation.Status,
				"mirror_evaluation_id": evaluation.ID,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrCourseEvaluationConflict
				}
				return err
			}
		}
		if err := s.courseRepoWithTx(tx).RecalculateCourseStats(evaluation.CourseID); err != nil {
			return err
		}
		teacherIDs := appendTeacherRefreshIDs(oldTeacherID, evaluation.TeacherID)
		for _, teacherID := range teacherIDs {
			if err := s.teacherRepoWithTx(tx).RecalculateTeacherStats(teacherID); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
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

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if evaluation.MirrorEvaluationID != nil {
			if err := tx.Delete(&model.TeacherEvaluations{}, *evaluation.MirrorEvaluationID).Error; err != nil {
				return err
			}
		}
		if err := tx.Delete(&model.CourseEvaluations{}, evaluationID).Error; err != nil {
			return err
		}
		if err := s.courseRepoWithTx(tx).RecalculateCourseStats(evaluation.CourseID); err != nil {
			return err
		}
		if evaluation.TeacherID != nil {
			return s.teacherRepoWithTx(tx).RecalculateTeacherStats(*evaluation.TeacherID)
		}
		return nil
	})
	if txErr != nil {
		return txErr
	}
	return nil
}

func (s *CourseService) ListMyCourseEvaluations(userID int64, page, size int) ([]repo.MyCourseEvaluationItem, int64, error) {
	fillPagination(&page, &size)
	return s.courseRepo.ListMyCourseEvaluations(userID, page, size)
}

func (s *CourseService) normalizeCourseEvaluationUsers(items []repo.CourseEvaluationItem) {
	for i := range items {
		if items[i].IsAnonymous {
			items[i].User = nil
		} else {
			items[i].User = &repo.UserBrief{
				ID:        items[i].AuthorID,
				Nickname:  items[i].AuthorName,
				AvatarURL: items[i].AuthorAvatarURL,
				Role:      externalUserRole(items[i].AuthorRole),
			}
		}
		for j := range items[i].Replies {
			if items[i].Replies[j].IsAnonymous {
				items[i].Replies[j].User = nil
			} else {
				items[i].Replies[j].User = &repo.UserBrief{
					ID:        items[i].Replies[j].UserID,
					Nickname:  items[i].Replies[j].AuthorName,
					AvatarURL: items[i].Replies[j].AuthorAvatar,
					Role:      externalUserRole(items[i].Replies[j].AuthorRole),
				}
			}
			if items[i].Replies[j].ReplyToUserID != nil {
				replyToUser := &repo.UserBrief{
					ID:       *items[i].Replies[j].ReplyToUserID,
					Nickname: items[i].Replies[j].ReplyToUserName,
					Role:     externalUserRole(items[i].Replies[j].ReplyToUserRole),
				}
				if items[i].Replies[j].ReplyToReplyID == nil {
					if items[i].IsAnonymous {
						replyToUser.ID = 0
						replyToUser.Nickname = "匿名用户"
						replyToUser.Role = ""
					}
				} else {
					for k := range items[i].Replies {
						if items[i].Replies[k].ID == *items[i].Replies[j].ReplyToReplyID && items[i].Replies[k].IsAnonymous {
							replyToUser.ID = 0
							replyToUser.Nickname = "匿名用户"
							replyToUser.Role = ""
							break
						}
					}
				}
				items[i].Replies[j].ReplyToUser = replyToUser
			}
		}
	}
}

func (s *CourseService) CreateCourseEvaluationReply(userID, evaluationID int64, content string, replyToReplyID, replyToUserID int64, isAnonymous bool) (*model.CourseEvaluationReplies, error) {
	evaluation, err := s.courseRepo.GetCourseEvaluationByID(evaluationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCourseEvaluationNotFound
	}
	if err != nil {
		return nil, err
	}

	var replyToReplyIDPtr *int64
	var replyToUserIDPtr *int64
	if replyToReplyID > 0 {
		reply, err := s.courseRepo.GetCourseEvaluationReplyByID(replyToReplyID)
		if errors.Is(err, gorm.ErrRecordNotFound) || reply.EvaluationID != evaluationID {
			return nil, ErrCourseEvaluationNotFound
		}
		if err != nil {
			return nil, err
		}
		replyToReplyIDPtr = &replyToReplyID
		if replyToUserID <= 0 {
			replyToUserID = reply.UserID
		}
	}
	if replyToReplyID == 0 && replyToUserID <= 0 {
		replyToUserID = evaluation.UserID
	}
	if replyToUserID > 0 {
		replyToUserIDPtr = &replyToUserID
	}

	reply := &model.CourseEvaluationReplies{
		EvaluationID:   evaluationID,
		UserID:         userID,
		Content:        content,
		IsAnonymous:    isAnonymous,
		ReplyToReplyID: replyToReplyIDPtr,
		ReplyToUserID:  replyToUserIDPtr,
	}

	recipientID := replyToUserID
	if recipientID == userID {
		recipientID = 0
	}

	if s.db != nil {
		err = s.db.Transaction(func(tx *gorm.DB) error {
			if err := s.courseRepoWithTx(tx).CreateCourseEvaluationReply(reply); err != nil {
				return err
			}
			if recipientID > 0 {
				notificationTargetType := model.LikeTargetTypeCourseEvaluation
				notificationTargetID := evaluationID
				if replyToReplyID > 0 {
					notificationTargetType = model.LikeTargetTypeCourseReply
					notificationTargetID = replyToReplyID
				}
				notification := buildEvaluationReplyNotification(
					recipientID,
					notificationTargetType,
					notificationTargetID,
					buildCourseEvaluationInteractionRoute(evaluation.CourseID, evaluationID, reply.ID),
				)
				if notification != nil {
					return tx.Create(notification).Error
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return reply, nil
	}

	if err := s.courseRepo.CreateCourseEvaluationReply(reply); err != nil {
		return nil, err
	}
	if recipientID > 0 {
		notificationTargetType := model.LikeTargetTypeCourseEvaluation
		notificationTargetID := evaluationID
		if replyToReplyID > 0 {
			notificationTargetType = model.LikeTargetTypeCourseReply
			notificationTargetID = replyToReplyID
		}
		notification := buildEvaluationReplyNotification(
			recipientID,
			notificationTargetType,
			notificationTargetID,
			buildCourseEvaluationInteractionRoute(evaluation.CourseID, evaluationID, reply.ID),
		)
		if notification != nil {
			if err := s.socialRepo.CreateNotification(notification); err != nil {
				return nil, err
			}
		}
	}
	return reply, nil
}

func (s *CourseService) GetCourseEvaluationReplyItem(id int64) (*repo.EvaluationReply, error) {
	reply, err := s.courseRepo.GetCourseEvaluationReplyDetailByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCourseEvaluationReplyNotFound
	}
	if err != nil {
		return nil, err
	}
	if !reply.IsAnonymous {
		reply.User = &repo.UserBrief{ID: reply.UserID, Nickname: reply.AuthorName, AvatarURL: reply.AuthorAvatar, Role: externalUserRole(reply.AuthorRole)}
	}
	if reply.ReplyToUserID != nil {
		replyToUser := &repo.UserBrief{ID: *reply.ReplyToUserID, Nickname: reply.ReplyToUserName, Role: externalUserRole(reply.ReplyToUserRole)}
		if reply.ReplyToReplyID == nil {
			evaluation, err := s.courseRepo.GetCourseEvaluationByID(reply.EvaluationID)
			if err == nil && evaluation.IsAnonymous {
				replyToUser.ID = 0
				replyToUser.Nickname = "匿名用户"
				replyToUser.Role = ""
			}
		} else {
			targetReply, err := s.courseRepo.GetCourseEvaluationReplyByID(*reply.ReplyToReplyID)
			if err == nil && targetReply.IsAnonymous {
				replyToUser.ID = 0
				replyToUser.Nickname = "匿名用户"
				replyToUser.Role = ""
			}
		}
		reply.ReplyToUser = replyToUser
	}
	return reply, nil
}

func (s *CourseService) UpdateCourseEvaluationReply(userID int64, replyID int64, content string, isAnonymous bool) (*model.CourseEvaluationReplies, error) {
	reply, err := s.courseRepo.GetCourseEvaluationReplyByID(replyID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCourseEvaluationReplyNotFound
	}
	if err != nil {
		return nil, err
	}
	if reply.UserID != userID {
		return nil, ErrCourseEvaluationReplyForbidden
	}
	reply.Content = content
	reply.IsAnonymous = isAnonymous
	if err := s.courseRepo.UpdateCourseEvaluationReply(reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (s *CourseService) DeleteCourseEvaluationReply(userID int64, userRole string, replyID int64) error {
	reply, err := s.courseRepo.GetCourseEvaluationReplyByID(replyID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrCourseEvaluationReplyNotFound
	}
	if err != nil {
		return err
	}
	if reply.UserID != userID && !isPrivilegedRole(userRole) {
		return ErrCourseEvaluationReplyForbidden
	}
	return s.courseRepo.DeleteCourseEvaluationReply(replyID)
}

func (s *CourseService) RandomShowCourses() (repo.RandomCourses, error) {
	// 先查缓存
	var randomCourses repo.RandomCourses
	result, err := utils.RDB.LRange(utils.Ctx, constant.CacheRandomCoursesPrefix+"666", 0, -1).Result()
	if err != nil {
		return randomCourses, err
	}
	if result != nil && len(result) == constant.RandCoursesCount {
		randomCourses.Items = make([]repo.RandomCourseItem, 0, len(result))
		for _, item := range result {
			var course repo.RandomCourseItem
			err := json.Unmarshal([]byte(item), &course)
			if err != nil {
				return repo.RandomCourses{}, err
			}
			randomCourses.Items = append(randomCourses.Items, course)
		}
		return randomCourses, nil
	}

	// 缓存没有换成查数据库
	ids, err := s.courseRepo.ListRandomCourseIDs(constant.RandCoursesCount)
	if err != nil {
		return repo.RandomCourses{}, err
	}
	if len(ids) == 0 {
		return repo.RandomCourses{}, nil
	}
	randomCourses, err = s.courseRepo.FindRandomCourses(ids)
	if err != nil {
		return repo.RandomCourses{}, err
	}

	return randomCourses, nil
}

func (s *CourseService) attachCourseRankingTeachers(items []repo.CourseRankingItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	teacherMap, err := s.courseRepo.ListTeacherBriefsByCourseIDs(ids)
	if err != nil {
		return err
	}
	for i := range items {
		items[i].Teachers = teacherMap[items[i].ID]
	}
	return nil
}

func (s *CourseService) validateCourseEvaluationRefs(courseID int64, teacherID *int64) error {
	exists, err := s.courseRepo.CourseExists(courseID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCourseDetailNotFound
	}
	if teacherID == nil {
		return nil
	}
	teacherExists, err := s.teacherRepo.TeacherExists(*teacherID)
	if err != nil {
		return err
	}
	if !teacherExists {
		return ErrTeacherNotFound
	}
	related, err := s.teacherRepo.TeacherCourseRelationExists(*teacherID, courseID)
	if err != nil {
		return err
	}
	if !related {
		return ErrTeacherCourseMismatch
	}
	return nil
}

func courseEvaluationMode(teacherID *int64) model.EvaluationMode {
	if teacherID == nil {
		return model.EvaluationModeSingle
	}
	return model.EvaluationModeLinked
}

func validateCourseEvaluationPayload(teacherID *int64, quality, grading, attendance *int) error {
	if teacherID == nil {
		return nil
	}
	if quality == nil || grading == nil || attendance == nil {
		return ErrEvaluationAssociationIncomplete
	}
	return nil
}

func (s *CourseService) hydrateCourseEvaluationItem(item *repo.CourseEvaluationItem, viewerID int64) error {
	if item.IsAnonymous {
		item.User = nil
	} else {
		item.User = &repo.UserBrief{
			ID:        item.AuthorID,
			Nickname:  item.AuthorName,
			AvatarURL: item.AuthorAvatarURL,
			Role:      externalUserRole(item.AuthorRole),
		}
	}
	for j := range item.Replies {
		item.Replies[j].User = &repo.UserBrief{
			ID:        item.Replies[j].UserID,
			Nickname:  item.Replies[j].AuthorName,
			AvatarURL: item.Replies[j].AuthorAvatar,
			Role:      externalUserRole(item.Replies[j].AuthorRole),
		}
		if item.Replies[j].ReplyToUserID != nil {
			item.Replies[j].ReplyToUser = &repo.UserBrief{
				ID:       *item.Replies[j].ReplyToUserID,
				Nickname: item.Replies[j].ReplyToUserName,
				Role:     externalUserRole(item.Replies[j].ReplyToUserRole),
			}
		}
	}
	if viewerID <= 0 || s.socialRepo == nil {
		return nil
	}
	evalLikes, err := s.socialRepo.ListLikedTargetIDs(viewerID, model.LikeTargetTypeCourseEvaluation, []int64{item.ID})
	if err != nil {
		return err
	}
	replyIDs := make([]int64, 0, len(item.Replies))
	for _, reply := range item.Replies {
		replyIDs = append(replyIDs, reply.ID)
	}
	replyLikes, err := s.socialRepo.ListLikedTargetIDs(viewerID, model.LikeTargetTypeCourseReply, replyIDs)
	if err != nil {
		return err
	}
	item.IsLiked = evalLikes[item.ID]
	for j := range item.Replies {
		item.Replies[j].IsLiked = replyLikes[item.Replies[j].ID]
	}
	return nil
}

func (s *CourseService) attachCourseEvaluationLikeState(items []repo.CourseEvaluationItem, viewerID int64) error {
	if viewerID <= 0 || s.socialRepo == nil || len(items) == 0 {
		return nil
	}
	evaluationIDs := make([]int64, 0, len(items))
	replyIDs := make([]int64, 0, len(items))
	for _, item := range items {
		evaluationIDs = append(evaluationIDs, item.ID)
		for _, reply := range item.Replies {
			replyIDs = append(replyIDs, reply.ID)
		}
	}

	evalLikes, err := s.socialRepo.ListLikedTargetIDs(viewerID, model.LikeTargetTypeCourseEvaluation, evaluationIDs)
	if err != nil {
		return err
	}
	replyLikes, err := s.socialRepo.ListLikedTargetIDs(viewerID, model.LikeTargetTypeCourseReply, replyIDs)
	if err != nil {
		return err
	}
	for i := range items {
		items[i].IsLiked = evalLikes[items[i].ID]
		for j := range items[i].Replies {
			items[i].Replies[j].IsLiked = replyLikes[items[i].Replies[j].ID]
		}
	}
	return nil
}

func (s *CourseService) tryRefreshCourseStats(courseID int64, reason string) {
	if s.courseRepo == nil || courseID <= 0 {
		return
	}
	if err := s.courseRepo.RecalculateCourseStats(courseID); err != nil {
		logger.Log.Error("课程聚合实时回刷失败",
			zap.Int64("course_id", courseID),
			zap.String("reason", reason),
			zap.Error(err))
	}
}

func (s *CourseService) tryRefreshTeacherStats(teacherID int64, reason string) {
	if s.teacherRepo == nil || teacherID <= 0 {
		return
	}
	if err := s.teacherRepo.RecalculateTeacherStats(teacherID); err != nil {
		logger.Log.Error("教师聚合实时回刷失败",
			zap.Int64("teacher_id", teacherID),
			zap.String("reason", reason),
			zap.Error(err))
	}
}

func (s *CourseService) courseRepoWithTx(tx *gorm.DB) repo.CourseRepository {
	withTx, ok := s.courseRepo.(interface {
		WithTx(*gorm.DB) repo.CourseRepository
	})
	if !ok {
		return s.courseRepo
	}
	return withTx.WithTx(tx)
}

func (s *CourseService) teacherRepoWithTx(tx *gorm.DB) repo.TeacherRepository {
	withTx, ok := s.teacherRepo.(interface {
		WithTx(*gorm.DB) repo.TeacherRepository
	})
	if !ok {
		return s.teacherRepo
	}
	return withTx.WithTx(tx)
}

func appendTeacherRefreshIDs(ids ...*int64) []int64 {
	result := make([]int64, 0, len(ids))
	seen := map[int64]struct{}{}
	for _, id := range ids {
		if id == nil {
			continue
		}
		if _, ok := seen[*id]; ok {
			continue
		}
		seen[*id] = struct{}{}
		result = append(result, *id)
	}
	return result
}

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
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrTeacherNotFound                 = errors.New("teacher not found")
	ErrCourseNotFound                  = errors.New("course not found")
	ErrTeacherCourseMismatch           = errors.New("teacher course mismatch")
	ErrTeacherEvaluationNotFound       = errors.New("teacher evaluation not found")
	ErrTeacherEvaluationConflict       = errors.New("teacher evaluation conflict")
	ErrTeacherEvaluationForbidden      = errors.New("teacher evaluation forbidden")
	ErrTeacherEvaluationReplyNotFound  = errors.New("teacher evaluation reply not found")
	ErrTeacherEvaluationReplyForbidden = errors.New("teacher evaluation reply forbidden")
	ErrEvaluationAssociationIncomplete = errors.New("evaluation association incomplete")
)

type TeacherService struct {
	db          *gorm.DB
	teacherRepo repo.TeacherRepository
	courseRepo  repo.CourseRepository
	socialRepo  repo.SocialRepository
}

func NewTeacherService(db *gorm.DB, tr repo.TeacherRepository, cr repo.CourseRepository, sr repo.SocialRepository) *TeacherService {
	return &TeacherService{db: db, teacherRepo: tr, courseRepo: cr, socialRepo: sr}
}

func (s *TeacherService) ListTeachers(query repo.TeacherListQuery) ([]repo.TeacherListItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.Sort == "" {
		query.Sort = "avg_score"
	}
	return s.teacherRepo.FindTeachers(query)
}

func (s *TeacherService) ListSimpleTeachers(q string) ([]repo.TeacherSimpleItem, error) {
	return s.teacherRepo.ListSimpleTeachers(q, 20)
}

func (s *TeacherService) GetTeacherDetail(id int64, viewerID int64) (*repo.TeacherDetail, error) {
	detail, err := s.teacherRepo.FindTeacherDetail(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTeacherNotFound
	}
	if err != nil {
		return nil, err
	}
	if viewerID > 0 {
		favorited, err := s.socialRepo.HasFavorite(viewerID, model.FavoriteTargetTypeTeacher, detail.ID)
		if err != nil {
			return nil, err
		}
		detail.IsFavorited = favorited
	}
	return detail, nil
}

func (s *TeacherService) ListTeacherRankings(query repo.TeacherRankingQuery) ([]repo.TeacherRankingItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	switch query.RankType {
	case "", "avg_score":
		query.RankType = "avg_score"
	case "avg_quality", "avg_grading", "avg_attendance", "favorite_count", "eval_count":
	default:
		query.RankType = "avg_score"
	}
	query.Period = "all"

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
			if err := s.attachTeacherRankingCourses(ordered); err != nil {
				return nil, 0, err
			}
			return ordered, total, nil
		}
	}

	items, total, err := s.teacherRepo.FindTeacherRankings(query)
	if err != nil {
		return nil, 0, err
	}
	if err := s.attachTeacherRankingCourses(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
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

	if err := s.attachTeacherEvaluationLikeState(items, userID); err != nil {
		return nil, 0, err
	}
	s.normalizeTeacherEvaluationUsers(items)

	return items, total, nil
}

func (s *TeacherService) CreateTeacherEvaluation(userID, teacherID int64, input model.TeacherEvaluations) (*model.TeacherEvaluations, error) {
	if err := s.validateTeacherEvaluationRefs(teacherID, input.CourseID); err != nil {
		return nil, err
	}
	if err := validateTeacherEvaluationPayload(input.CourseID, input.HomeworkScore, input.GainScore, input.ExamDifficultyScore); err != nil {
		return nil, err
	}
	if _, err := s.teacherRepo.FindTeacherEvaluationByContext(userID, teacherID, input.CourseID, teacherEvaluationMode(input.CourseID)); err == nil {
		return nil, ErrTeacherEvaluationConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	input.UserID = userID
	input.TeacherID = teacherID
	input.Status = model.ResourceStatusApproved
	input.Mode = teacherEvaluationMode(input.CourseID)
	if input.Mode == model.EvaluationModeLinked {
		input.MirrorEntityType = model.MirrorEntityTypeCourse
	} else {
		input.MirrorEntityType = ""
	}
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&input).Error; err != nil {
			if isDuplicateKeyErr(err) {
				return ErrTeacherEvaluationConflict
			}
			return err
		}
		if input.Mode != model.EvaluationModeLinked {
			if err := s.teacherRepoWithTx(tx).RecalculateTeacherStats(teacherID); err != nil {
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
		courseEval := &model.CourseEvaluations{
			UserID:           userID,
			CourseID:         *input.CourseID,
			TeacherID:        &teacherID,
			Mode:             model.EvaluationModeLinked,
			MirrorEntityType: model.MirrorEntityTypeTeacher,
			WorkloadScore:    valueOrZero(input.HomeworkScore),
			GainScore:        valueOrZero(input.GainScore),
			DifficultyScore:  valueOrZero(input.ExamDifficultyScore),
			TeachingScore:    intPtr(input.TeachingScore),
			GradingScore:     intPtr(input.GradingScore),
			AttendanceScore:  intPtr(input.AttendanceScore),
			Comment:          input.Comment,
			IsAnonymous:      input.IsAnonymous,
			Status:           model.ResourceStatusApproved,
		}
		if err := tx.Create(courseEval).Error; err != nil {
			if isDuplicateKeyErr(err) {
				return ErrTeacherEvaluationConflict
			}
			return err
		}
		input.MirrorEvaluationID = &courseEval.ID
		courseEval.MirrorEvaluationID = &input.ID
		if err := tx.Model(&model.TeacherEvaluations{}).Where("id = ?", input.ID).
			Updates(map[string]interface{}{
				"mirror_evaluation_id": input.MirrorEvaluationID,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.CourseEvaluations{}).Where("id = ?", courseEval.ID).
			Updates(map[string]interface{}{
				"mirror_evaluation_id": courseEval.MirrorEvaluationID,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
			return err
		}
		if err := s.teacherRepoWithTx(tx).RecalculateTeacherStats(teacherID); err != nil {
			return err
		}
		if err := s.courseRepoWithTx(tx).RecalculateCourseStats(*input.CourseID); err != nil {
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

func (s *TeacherService) GetTeacherEvaluationItem(id int64, viewerID int64) (*repo.TeacherEvaluationItem, error) {
	item, err := s.teacherRepo.GetTeacherEvaluationItemByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTeacherEvaluationNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.hydrateTeacherEvaluationItem(item, viewerID); err != nil {
		return nil, err
	}
	return item, nil
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
	if err := validateTeacherEvaluationPayload(input.CourseID, input.HomeworkScore, input.GainScore, input.ExamDifficultyScore); err != nil {
		return nil, err
	}

	oldCourseID := evaluation.CourseID
	oldMirrorID := evaluation.MirrorEvaluationID
	evaluation.CourseID = input.CourseID
	evaluation.Mode = teacherEvaluationMode(input.CourseID)
	if evaluation.Mode == model.EvaluationModeLinked {
		evaluation.MirrorEntityType = model.MirrorEntityTypeCourse
	} else {
		evaluation.MirrorEntityType = ""
	}
	evaluation.TeachingScore = input.TeachingScore
	evaluation.GradingScore = input.GradingScore
	evaluation.AttendanceScore = input.AttendanceScore
	evaluation.HomeworkScore = input.HomeworkScore
	evaluation.GainScore = input.GainScore
	evaluation.ExamDifficultyScore = input.ExamDifficultyScore
	evaluation.Comment = input.Comment
	evaluation.IsAnonymous = input.IsAnonymous
	evaluation.Status = model.ResourceStatusApproved
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		switch {
		case oldMirrorID == nil && evaluation.Mode == model.EvaluationModeSingle:
			if err := tx.Model(&model.TeacherEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"course_id":            evaluation.CourseID,
				"mode":                 evaluation.Mode,
				"mirror_evaluation_id": nil,
				"mirror_entity_type":   nil,
				"teaching_score":       evaluation.TeachingScore,
				"grading_score":        evaluation.GradingScore,
				"attendance_score":     evaluation.AttendanceScore,
				"workload_score":       evaluation.HomeworkScore,
				"gain_score":           evaluation.GainScore,
				"difficulty_score":     evaluation.ExamDifficultyScore,
				"comment":              evaluation.Comment,
				"is_anonymous":         evaluation.IsAnonymous,
				"status":               evaluation.Status,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrTeacherEvaluationConflict
				}
				return err
			}
		case oldMirrorID == nil && evaluation.Mode == model.EvaluationModeLinked:
			mirror := &model.CourseEvaluations{
				UserID:           evaluation.UserID,
				CourseID:         *evaluation.CourseID,
				TeacherID:        int64Ptr(evaluation.TeacherID),
				Mode:             model.EvaluationModeLinked,
				MirrorEntityType: model.MirrorEntityTypeTeacher,
				WorkloadScore:    valueOrZero(evaluation.HomeworkScore),
				GainScore:        valueOrZero(evaluation.GainScore),
				DifficultyScore:  valueOrZero(evaluation.ExamDifficultyScore),
				TeachingScore:    intPtr(evaluation.TeachingScore),
				GradingScore:     intPtr(evaluation.GradingScore),
				AttendanceScore:  intPtr(evaluation.AttendanceScore),
				Comment:          evaluation.Comment,
				IsAnonymous:      evaluation.IsAnonymous,
				Status:           evaluation.Status,
			}
			if err := tx.Model(&model.TeacherEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"course_id":          evaluation.CourseID,
				"mode":               evaluation.Mode,
				"mirror_entity_type": evaluation.MirrorEntityType,
				"teaching_score":     evaluation.TeachingScore,
				"grading_score":      evaluation.GradingScore,
				"attendance_score":   evaluation.AttendanceScore,
				"workload_score":     evaluation.HomeworkScore,
				"gain_score":         evaluation.GainScore,
				"difficulty_score":   evaluation.ExamDifficultyScore,
				"comment":            evaluation.Comment,
				"is_anonymous":       evaluation.IsAnonymous,
				"status":             evaluation.Status,
				"updated_at":         gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrTeacherEvaluationConflict
				}
				return err
			}
			if err := tx.Create(mirror).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrTeacherEvaluationConflict
				}
				return err
			}
			evaluation.MirrorEvaluationID = &mirror.ID
			if err := tx.Model(&model.TeacherEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"mirror_evaluation_id": evaluation.MirrorEvaluationID,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.CourseEvaluations{}).Where("id = ?", mirror.ID).Updates(map[string]interface{}{
				"mirror_evaluation_id": evaluation.ID,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				return err
			}
		case oldMirrorID != nil && evaluation.Mode == model.EvaluationModeSingle:
			if err := tx.Delete(&model.CourseEvaluations{}, *oldMirrorID).Error; err != nil {
				return err
			}
			evaluation.MirrorEvaluationID = nil
			if err := tx.Model(&model.TeacherEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"course_id":            nil,
				"mode":                 evaluation.Mode,
				"mirror_evaluation_id": nil,
				"teaching_score":       evaluation.TeachingScore,
				"grading_score":        evaluation.GradingScore,
				"attendance_score":     evaluation.AttendanceScore,
				"workload_score":       nil,
				"gain_score":           nil,
				"difficulty_score":     nil,
				"comment":              evaluation.Comment,
				"is_anonymous":         evaluation.IsAnonymous,
				"status":               evaluation.Status,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				return err
			}
		default:
			if err := tx.Model(&model.TeacherEvaluations{}).Where("id = ?", evaluation.ID).Updates(map[string]interface{}{
				"course_id":        evaluation.CourseID,
				"mode":             evaluation.Mode,
				"teaching_score":   evaluation.TeachingScore,
				"grading_score":    evaluation.GradingScore,
				"attendance_score": evaluation.AttendanceScore,
				"workload_score":   evaluation.HomeworkScore,
				"gain_score":       evaluation.GainScore,
				"difficulty_score": evaluation.ExamDifficultyScore,
				"comment":          evaluation.Comment,
				"is_anonymous":     evaluation.IsAnonymous,
				"status":           evaluation.Status,
				"updated_at":       gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrTeacherEvaluationConflict
				}
				return err
			}
			if err := tx.Model(&model.CourseEvaluations{}).Where("id = ?", *oldMirrorID).Updates(map[string]interface{}{
				"course_id":            *evaluation.CourseID,
				"teacher_id":           evaluation.TeacherID,
				"mode":                 evaluation.Mode,
				"mirror_entity_type":   model.MirrorEntityTypeTeacher,
				"teaching_score":       evaluation.TeachingScore,
				"grading_score":        evaluation.GradingScore,
				"attendance_score":     evaluation.AttendanceScore,
				"workload_score":       valueOrZero(evaluation.HomeworkScore),
				"gain_score":           valueOrZero(evaluation.GainScore),
				"difficulty_score":     valueOrZero(evaluation.ExamDifficultyScore),
				"comment":              evaluation.Comment,
				"is_anonymous":         evaluation.IsAnonymous,
				"status":               evaluation.Status,
				"mirror_evaluation_id": evaluation.ID,
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
				if isDuplicateKeyErr(err) {
					return ErrTeacherEvaluationConflict
				}
				return err
			}
		}
		if err := s.teacherRepoWithTx(tx).RecalculateTeacherStats(evaluation.TeacherID); err != nil {
			return err
		}
		courseIDs := appendCourseRefreshIDs(oldCourseID, evaluation.CourseID)
		for _, courseID := range courseIDs {
			if err := s.courseRepoWithTx(tx).RecalculateCourseStats(courseID); err != nil {
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

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if evaluation.MirrorEvaluationID != nil {
			if err := tx.Delete(&model.CourseEvaluations{}, *evaluation.MirrorEvaluationID).Error; err != nil {
				return err
			}
		}
		if err := tx.Delete(&model.TeacherEvaluations{}, evaluationID).Error; err != nil {
			return err
		}
		if err := s.teacherRepoWithTx(tx).RecalculateTeacherStats(evaluation.TeacherID); err != nil {
			return err
		}
		if evaluation.CourseID != nil {
			return s.courseRepoWithTx(tx).RecalculateCourseStats(*evaluation.CourseID)
		}
		return nil
	})
	if txErr != nil {
		return txErr
	}
	return nil
}

func (s *TeacherService) ListMyTeacherEvaluations(userID int64, page, size int) ([]repo.MyTeacherEvaluationItem, int64, error) {
	fillPagination(&page, &size)
	return s.teacherRepo.ListMyTeacherEvaluations(userID, page, size)
}

func (s *TeacherService) normalizeTeacherEvaluationUsers(items []repo.TeacherEvaluationItem) {
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

func (s *TeacherService) CreateTeacherEvaluationReply(userID, evaluationID int64, content string, replyToReplyID, replyToUserID int64, isAnonymous bool) (*model.TeacherEvaluationReplies, error) {
	evaluation, err := s.teacherRepo.GetTeacherEvaluationByID(evaluationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTeacherEvaluationNotFound
	}
	if err != nil {
		return nil, err
	}

	var replyToReplyIDPtr *int64
	var replyToUserIDPtr *int64
	if replyToReplyID > 0 {
		reply, err := s.teacherRepo.GetTeacherEvaluationReplyByID(replyToReplyID)
		if errors.Is(err, gorm.ErrRecordNotFound) || reply.EvaluationID != evaluationID {
			return nil, ErrTeacherEvaluationNotFound
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

	reply := &model.TeacherEvaluationReplies{
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
			if err := s.teacherRepoWithTx(tx).CreateTeacherEvaluationReply(reply); err != nil {
				return err
			}
			if recipientID > 0 {
				notificationTargetType := model.LikeTargetTypeTeacherEvaluation
				notificationTargetID := evaluationID
				if replyToReplyID > 0 {
					notificationTargetType = model.LikeTargetTypeTeacherReply
					notificationTargetID = replyToReplyID
				}
				notification := buildEvaluationReplyNotification(
					recipientID,
					notificationTargetType,
					notificationTargetID,
					buildTeacherEvaluationInteractionRoute(evaluation.TeacherID, evaluationID, reply.ID),
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

	if err := s.teacherRepo.CreateTeacherEvaluationReply(reply); err != nil {
		return nil, err
	}
	if recipientID > 0 {
		notificationTargetType := model.LikeTargetTypeTeacherEvaluation
		notificationTargetID := evaluationID
		if replyToReplyID > 0 {
			notificationTargetType = model.LikeTargetTypeTeacherReply
			notificationTargetID = replyToReplyID
		}
		notification := buildEvaluationReplyNotification(
			recipientID,
			notificationTargetType,
			notificationTargetID,
			buildTeacherEvaluationInteractionRoute(evaluation.TeacherID, evaluationID, reply.ID),
		)
		if notification != nil {
			if err := s.socialRepo.CreateNotification(notification); err != nil {
				return nil, err
			}
		}
	}
	return reply, nil
}

func (s *TeacherService) GetTeacherEvaluationReplyItem(id int64) (*repo.EvaluationReply, error) {
	reply, err := s.teacherRepo.GetTeacherEvaluationReplyDetailByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTeacherEvaluationReplyNotFound
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
			evaluation, err := s.teacherRepo.GetTeacherEvaluationByID(reply.EvaluationID)
			if err == nil && evaluation.IsAnonymous {
				replyToUser.ID = 0
				replyToUser.Nickname = "匿名用户"
				replyToUser.Role = ""
			}
		} else {
			targetReply, err := s.teacherRepo.GetTeacherEvaluationReplyByID(*reply.ReplyToReplyID)
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

func (s *TeacherService) UpdateTeacherEvaluationReply(userID int64, replyID int64, content string, isAnonymous bool) (*model.TeacherEvaluationReplies, error) {
	reply, err := s.teacherRepo.GetTeacherEvaluationReplyByID(replyID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTeacherEvaluationReplyNotFound
	}
	if err != nil {
		return nil, err
	}
	if reply.UserID != userID {
		return nil, ErrTeacherEvaluationReplyForbidden
	}
	reply.Content = content
	reply.IsAnonymous = isAnonymous
	if err := s.teacherRepo.UpdateTeacherEvaluationReply(reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (s *TeacherService) DeleteTeacherEvaluationReply(userID int64, userRole string, replyID int64) error {
	reply, err := s.teacherRepo.GetTeacherEvaluationReplyByID(replyID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrTeacherEvaluationReplyNotFound
	}
	if err != nil {
		return err
	}
	if reply.UserID != userID && !isPrivilegedRole(userRole) {
		return ErrTeacherEvaluationReplyForbidden
	}
	return s.teacherRepo.DeleteTeacherEvaluationReply(replyID)
}

func (s *TeacherService) validateTeacherEvaluationRefs(teacherID int64, courseID *int64) error {
	teacherExists, err := s.teacherRepo.TeacherExists(teacherID)
	if err != nil {
		return err
	}
	if !teacherExists {
		return ErrTeacherNotFound
	}
	if courseID == nil {
		return nil
	}

	courseExists, err := s.teacherRepo.CourseExists(*courseID)
	if err != nil {
		return err
	}
	if !courseExists {
		return ErrCourseNotFound
	}

	related, err := s.teacherRepo.TeacherCourseRelationExists(teacherID, *courseID)
	if err != nil {
		return err
	}
	if !related {
		return ErrTeacherCourseMismatch
	}
	return nil
}

func (s *TeacherService) RandomShowTeachers() (repo.RandomTeachers, error) {
	var randomTeachers repo.RandomTeachers
	// 先查缓存
	result, err := utils.RDB.LRange(utils.Ctx, constant.CacheRandomTeachersPrefix+"666", 0, -1).Result()
	if err != nil {
		return randomTeachers, nil
	}
	if result != nil && len(result) == constant.RandTeachersCount {
		for _, item := range result {
			var teacherItem repo.RandomTeacherItem
			err := json.Unmarshal([]byte(item), &teacherItem)
			if err != nil {
				return repo.RandomTeachers{}, err
			}
			randomTeachers.Items = append(randomTeachers.Items, teacherItem)
		}
		return randomTeachers, nil
	}

	// 缓存没有查数据库
	ids := utils.RandUniqueInts(constant.RandTeachersMin, constant.RandTeachersMax, constant.RandTeachersCount)
	if ids == nil || len(ids) == 0 {
		ids = []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	}
	randomTeachers, err = s.teacherRepo.FindRandomTeachers(ids)
	if err != nil {
		return repo.RandomTeachers{}, err
	}

	return randomTeachers, nil
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

func teacherEvaluationMode(courseID *int64) model.EvaluationMode {
	if courseID == nil {
		return model.EvaluationModeSingle
	}
	return model.EvaluationModeLinked
}

func validateTeacherEvaluationPayload(courseID *int64, homework, gain, exam *int) error {
	if courseID == nil {
		return nil
	}
	if homework == nil || gain == nil || exam == nil {
		return ErrEvaluationAssociationIncomplete
	}
	return nil
}

func (s *TeacherService) hydrateTeacherEvaluationItem(item *repo.TeacherEvaluationItem, viewerID int64) error {
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
	evalLikes, err := s.socialRepo.ListLikedTargetIDs(viewerID, model.LikeTargetTypeTeacherEvaluation, []int64{item.ID})
	if err != nil {
		return err
	}
	replyIDs := make([]int64, 0, len(item.Replies))
	for _, reply := range item.Replies {
		replyIDs = append(replyIDs, reply.ID)
	}
	replyLikes, err := s.socialRepo.ListLikedTargetIDs(viewerID, model.LikeTargetTypeTeacherReply, replyIDs)
	if err != nil {
		return err
	}
	item.IsLiked = evalLikes[item.ID]
	for j := range item.Replies {
		item.Replies[j].IsLiked = replyLikes[item.Replies[j].ID]
	}
	return nil
}

func (s *TeacherService) attachTeacherEvaluationLikeState(items []repo.TeacherEvaluationItem, viewerID int64) error {
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

	evalLikes, err := s.socialRepo.ListLikedTargetIDs(viewerID, model.LikeTargetTypeTeacherEvaluation, evaluationIDs)
	if err != nil {
		return err
	}
	replyLikes, err := s.socialRepo.ListLikedTargetIDs(viewerID, model.LikeTargetTypeTeacherReply, replyIDs)
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

func (s *TeacherService) tryRefreshTeacherStats(teacherID int64, reason string) {
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

func (s *TeacherService) tryRefreshCourseStats(courseID int64, reason string) {
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

func (s *TeacherService) teacherRepoWithTx(tx *gorm.DB) repo.TeacherRepository {
	withTx, ok := s.teacherRepo.(interface {
		WithTx(*gorm.DB) repo.TeacherRepository
	})
	if !ok {
		return s.teacherRepo
	}
	return withTx.WithTx(tx)
}

func (s *TeacherService) courseRepoWithTx(tx *gorm.DB) repo.CourseRepository {
	withTx, ok := s.courseRepo.(interface {
		WithTx(*gorm.DB) repo.CourseRepository
	})
	if !ok {
		return s.courseRepo
	}
	return withTx.WithTx(tx)
}

func appendCourseRefreshIDs(ids ...*int64) []int64 {
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

func (s *TeacherService) attachTeacherRankingCourses(items []repo.TeacherRankingItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	courseMap, err := s.teacherRepo.ListCourseBriefsByTeacherIDs(ids)
	if err != nil {
		return err
	}
	for i := range items {
		items[i].Courses = courseMap[items[i].ID]
	}
	return nil
}

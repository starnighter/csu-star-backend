package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrSocialTargetNotFound = errors.New("social target not found")
	ErrAlreadyLiked         = errors.New("already liked")
	ErrAlreadyFavorited     = errors.New("already favorited")
)

type SocialService struct {
	db           *gorm.DB
	socialRepo   repo.SocialRepository
	courseRepo   repo.CourseRepository
	teacherRepo  repo.TeacherRepository
	resourceRepo repo.ResourceRepository
	commentRepo  repo.CommentRepository
}

func NewSocialService(
	db *gorm.DB,
	sr repo.SocialRepository,
	cr repo.CourseRepository,
	tr repo.TeacherRepository,
	rr repo.ResourceRepository,
	comr repo.CommentRepository,
) *SocialService {
	return &SocialService{
		db:           db,
		socialRepo:   sr,
		courseRepo:   cr,
		teacherRepo:  tr,
		resourceRepo: rr,
		commentRepo:  comr,
	}
}

func (s *SocialService) Like(userID int64, targetType model.LikeTargetType, targetID int64) error {
	ok, err := s.likeTargetExists(targetType, targetID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSocialTargetNotFound
	}

	recipientID, err := s.socialRepo.GetLikeNotificationRecipient(targetType, targetID)
	if err != nil {
		return err
	}

	var notification *model.Notifications
	if recipientID > 0 && recipientID != userID {
		route, err := s.resolveInteractionRoute(targetType, targetID)
		if err != nil {
			return err
		}
		notification = &model.Notifications{
			UserID:    recipientID,
			Type:      model.NotificationTypeLiked,
			Category:  model.NotificationCategoryInteraction,
			Result:    model.NotificationResultInform,
			Title:     "收到新的点赞",
			Content:   buildLikeNotificationContent(targetType),
			RelatedID: targetID,
			IsRead:    false,
			IsGlobal:  false,
			Metadata:  buildInteractionMetadata(string(targetType), targetID, route),
		}
	}

	return s.withWriteTx(func(socialRepo repo.SocialRepository, courseRepo repo.CourseRepository, _ repo.TeacherRepository, resourceRepo repo.ResourceRepository) error {
		if err := socialRepo.CreateLikeWithEffects(&model.Likes{
			UserID:     userID,
			TargetType: targetType,
			TargetID:   targetID,
		}, recipientID, notification); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return ErrAlreadyLiked
			}
			return err
		}
		if targetType == model.LikeTargetTypeResource {
			courseID, err := resourceRepo.GetResourceCourseID(targetID)
			if err != nil {
				return err
			}
			if err := courseRepo.AdjustCourseAggregates(courseID, 0, 0, 0, 1, 0, 0, 0); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SocialService) Unlike(userID int64, targetType model.LikeTargetType, targetID int64) error {
	return s.withWriteTx(func(socialRepo repo.SocialRepository, courseRepo repo.CourseRepository, _ repo.TeacherRepository, resourceRepo repo.ResourceRepository) error {
		if err := socialRepo.DeleteLikeWithEffects(userID, targetType, targetID); err != nil {
			return err
		}
		if targetType == model.LikeTargetTypeResource {
			courseID, err := resourceRepo.GetResourceCourseID(targetID)
			if err != nil {
				return err
			}
			if err := courseRepo.AdjustCourseAggregates(courseID, 0, 0, 0, -1, 0, 0, 0); err != nil {
				return err
			}
		}
		return nil
	})
}

func buildLikeNotificationContent(targetType model.LikeTargetType) string {
	switch targetType {
	case model.LikeTargetTypeResource:
		return "你的资源收到了新的点赞。"
	case model.LikeTargetTypeTeacherEvaluation:
		return "你的教师评价收到了新的点赞。"
	case model.LikeTargetTypeCourseEvaluation:
		return "你的课程评价收到了新的点赞。"
	case model.LikeTargetTypeTeacherReply:
		return "你的教师评价回复收到了新的点赞。"
	case model.LikeTargetTypeCourseReply:
		return "你的课程评价回复收到了新的点赞。"
	case model.LikeTargetTypeComment:
		return "你的评论收到了新的点赞。"
	default:
		return "你收到了新的点赞。"
	}
}

func (s *SocialService) Favorite(userID int64, targetType model.FavoriteTargetType, targetID int64) error {
	ok, err := s.favoriteTargetExists(targetType, targetID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSocialTargetNotFound
	}

	var notification *model.Notifications
	if targetType == model.FavoriteTargetTypeResource {
		recipientID, err := s.socialRepo.GetResourceOwnerID(targetID)
		if err != nil {
			return err
		}
		if recipientID > 0 && recipientID != userID {
			notification = buildResourceFavoriteNotification(recipientID, targetID)
		}
	}

	return s.withWriteTx(func(socialRepo repo.SocialRepository, courseRepo repo.CourseRepository, teacherRepo repo.TeacherRepository, resourceRepo repo.ResourceRepository) error {
		if err := socialRepo.CreateFavorite(&model.Favorites{
			UserID:     userID,
			TargetType: targetType,
			TargetID:   targetID,
		}); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return ErrAlreadyFavorited
			}
			return err
		}
		if notification != nil {
			if err := socialRepo.CreateNotification(notification); err != nil {
				return err
			}
		}
		switch targetType {
		case model.FavoriteTargetTypeResource:
			if courseRepo != nil && resourceRepo != nil {
				courseID, err := resourceRepo.GetResourceCourseID(targetID)
				if err != nil {
					return err
				}
				if err := courseRepo.AdjustCourseAggregates(courseID, 0, 0, 0, 0, 0, 1, 0); err != nil {
					return err
				}
			}
		case model.FavoriteTargetTypeCourse:
			if courseRepo != nil {
				if err := courseRepo.AdjustCourseAggregates(targetID, 0, 0, 0, 0, 1, 0, 0); err != nil {
					return err
				}
			}
		case model.FavoriteTargetTypeTeacher:
			if teacherRepo != nil {
				if err := teacherRepo.AdjustTeacherAggregates(targetID, 1, 0); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *SocialService) resolveInteractionRoute(targetType model.LikeTargetType, targetID int64) (*interactionRoute, error) {
	switch targetType {
	case model.LikeTargetTypeResource:
		return buildResourceInteractionRoute(targetID), nil
	case model.LikeTargetTypeTeacherEvaluation:
		evaluation, err := s.teacherRepo.GetTeacherEvaluationByID(targetID)
		if err != nil {
			return nil, err
		}
		return buildTeacherEvaluationInteractionRoute(evaluation.TeacherID, evaluation.ID, 0), nil
	case model.LikeTargetTypeCourseEvaluation:
		evaluation, err := s.courseRepo.GetCourseEvaluationByID(targetID)
		if err != nil {
			return nil, err
		}
		return buildCourseEvaluationInteractionRoute(evaluation.CourseID, evaluation.ID, 0), nil
	case model.LikeTargetTypeTeacherReply:
		reply, err := s.teacherRepo.GetTeacherEvaluationReplyByID(targetID)
		if err != nil {
			return nil, err
		}
		evaluation, err := s.teacherRepo.GetTeacherEvaluationByID(reply.EvaluationID)
		if err != nil {
			return nil, err
		}
		return buildTeacherEvaluationInteractionRoute(evaluation.TeacherID, reply.EvaluationID, reply.ID), nil
	case model.LikeTargetTypeCourseReply:
		reply, err := s.courseRepo.GetCourseEvaluationReplyByID(targetID)
		if err != nil {
			return nil, err
		}
		evaluation, err := s.courseRepo.GetCourseEvaluationByID(reply.EvaluationID)
		if err != nil {
			return nil, err
		}
		return buildCourseEvaluationInteractionRoute(evaluation.CourseID, reply.EvaluationID, reply.ID), nil
	case model.LikeTargetTypeComment:
		if s.commentRepo == nil {
			return nil, nil
		}
		comment, err := s.commentRepo.GetCommentByID(targetID)
		if err != nil {
			return nil, err
		}
		if comment.TargetType != model.CommentTargetTypeResource {
			return nil, nil
		}
		commentID := comment.ID
		replyID := int64(0)
		if comment.ParentID != nil && *comment.ParentID > 0 {
			commentID = *comment.ParentID
			replyID = comment.ID
		}
		return &interactionRoute{
			targetPage: "resource",
			targetID:   comment.TargetID,
			commentID:  commentID,
			replyID:    replyID,
		}, nil
	default:
		return nil, nil
	}
}

func (s *SocialService) Unfavorite(userID int64, targetType model.FavoriteTargetType, targetID int64) error {
	return s.withWriteTx(func(socialRepo repo.SocialRepository, courseRepo repo.CourseRepository, teacherRepo repo.TeacherRepository, resourceRepo repo.ResourceRepository) error {
		hasFavorite, err := socialRepo.HasFavorite(userID, targetType, targetID)
		if err != nil {
			return err
		}
		if !hasFavorite {
			return nil
		}
		if err := socialRepo.DeleteFavorite(userID, targetType, targetID); err != nil {
			return err
		}
		switch targetType {
		case model.FavoriteTargetTypeResource:
			if courseRepo != nil && resourceRepo != nil {
				courseID, err := resourceRepo.GetResourceCourseID(targetID)
				if err != nil {
					return err
				}
				if err := courseRepo.AdjustCourseAggregates(courseID, 0, 0, 0, 0, 0, -1, 0); err != nil {
					return err
				}
			}
		case model.FavoriteTargetTypeCourse:
			if courseRepo != nil {
				if err := courseRepo.AdjustCourseAggregates(targetID, 0, 0, 0, 0, -1, 0, 0); err != nil {
					return err
				}
			}
		case model.FavoriteTargetTypeTeacher:
			if teacherRepo != nil {
				if err := teacherRepo.AdjustTeacherAggregates(targetID, -1, 0); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *SocialService) likeTargetExists(targetType model.LikeTargetType, targetID int64) (bool, error) {
	switch targetType {
	case model.LikeTargetTypeResource:
		return s.socialRepo.ResourceExists(targetID)
	case model.LikeTargetTypeTeacherEvaluation:
		return s.socialRepo.TeacherEvaluationExists(targetID)
	case model.LikeTargetTypeCourseEvaluation:
		return s.socialRepo.CourseEvaluationExists(targetID)
	case model.LikeTargetTypeTeacherReply:
		return s.socialRepo.TeacherEvaluationReplyExists(targetID)
	case model.LikeTargetTypeCourseReply:
		return s.socialRepo.CourseEvaluationReplyExists(targetID)
	case model.LikeTargetTypeComment:
		return s.socialRepo.CommentExists(targetID)
	default:
		return false, nil
	}
}

func (s *SocialService) favoriteTargetExists(targetType model.FavoriteTargetType, targetID int64) (bool, error) {
	switch targetType {
	case model.FavoriteTargetTypeResource:
		return s.socialRepo.ResourceExists(targetID)
	case model.FavoriteTargetTypeTeacher:
		return s.socialRepo.TeacherExists(targetID)
	case model.FavoriteTargetTypeCourse:
		return s.socialRepo.CourseExists(targetID)
	default:
		return false, nil
	}
}

func (s *SocialService) withWriteTx(fn func(repo.SocialRepository, repo.CourseRepository, repo.TeacherRepository, repo.ResourceRepository) error) error {
	if s.db == nil {
		return fn(s.socialRepo, s.courseRepo, s.teacherRepo, s.resourceRepo)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return fn(
			s.socialRepoWithTx(tx),
			s.courseRepoWithTx(tx),
			s.teacherRepoWithTx(tx),
			s.resourceRepoWithTx(tx),
		)
	})
}

func (s *SocialService) socialRepoWithTx(tx *gorm.DB) repo.SocialRepository {
	withTx, ok := s.socialRepo.(interface {
		WithTx(*gorm.DB) repo.SocialRepository
	})
	if !ok {
		return s.socialRepo
	}
	return withTx.WithTx(tx)
}

func (s *SocialService) courseRepoWithTx(tx *gorm.DB) repo.CourseRepository {
	withTx, ok := s.courseRepo.(interface {
		WithTx(*gorm.DB) repo.CourseRepository
	})
	if !ok {
		return s.courseRepo
	}
	return withTx.WithTx(tx)
}

func (s *SocialService) teacherRepoWithTx(tx *gorm.DB) repo.TeacherRepository {
	withTx, ok := s.teacherRepo.(interface {
		WithTx(*gorm.DB) repo.TeacherRepository
	})
	if !ok {
		return s.teacherRepo
	}
	return withTx.WithTx(tx)
}

func (s *SocialService) resourceRepoWithTx(tx *gorm.DB) repo.ResourceRepository {
	withTx, ok := s.resourceRepo.(interface {
		WithTx(*gorm.DB) repo.ResourceRepository
	})
	if !ok {
		return s.resourceRepo
	}
	return withTx.WithTx(tx)
}

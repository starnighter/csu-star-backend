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
	socialRepo repo.SocialRepository
}

func NewSocialService(sr repo.SocialRepository) *SocialService {
	return &SocialService{socialRepo: sr}
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
		notification = &model.Notifications{
			UserID:    recipientID,
			Type:      model.NotificationTypeLiked,
			Title:     "收到新的点赞",
			Content:   buildLikeNotificationContent(targetType),
			RelatedID: targetID,
			IsRead:    false,
			IsGlobal:  false,
		}
	}

	if err := s.socialRepo.CreateLikeWithEffects(&model.Likes{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
	}, recipientID, notification); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return ErrAlreadyLiked
		}
		return err
	}
	return nil
}

func (s *SocialService) Unlike(userID int64, targetType model.LikeTargetType, targetID int64) error {
	return s.socialRepo.DeleteLikeWithEffects(userID, targetType, targetID)
}

func buildLikeNotificationContent(targetType model.LikeTargetType) string {
	switch targetType {
	case model.LikeTargetTypeResource:
		return "你的资源收到了新的点赞。"
	case model.LikeTargetTypeTeacherEvaluation:
		return "你的教师评价收到了新的点赞。"
	case model.LikeTargetTypeCourseEvaluation:
		return "你的课程评价收到了新的点赞。"
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

	if err := s.socialRepo.CreateFavorite(&model.Favorites{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return ErrAlreadyFavorited
		}
		return err
	}
	return nil
}

func (s *SocialService) Unfavorite(userID int64, targetType model.FavoriteTargetType, targetID int64) error {
	hasFavorite, err := s.socialRepo.HasFavorite(userID, targetType, targetID)
	if err != nil {
		return err
	}
	if !hasFavorite {
		return nil
	}
	return s.socialRepo.DeleteFavorite(userID, targetType, targetID)
}

func (s *SocialService) likeTargetExists(targetType model.LikeTargetType, targetID int64) (bool, error) {
	switch targetType {
	case model.LikeTargetTypeResource:
		return s.socialRepo.ResourceExists(targetID)
	case model.LikeTargetTypeTeacherEvaluation:
		return s.socialRepo.TeacherEvaluationExists(targetID)
	case model.LikeTargetTypeCourseEvaluation:
		return s.socialRepo.CourseEvaluationExists(targetID)
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

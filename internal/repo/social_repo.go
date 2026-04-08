package repo

import (
	"csu-star-backend/internal/model"

	"gorm.io/gorm"
)

type SocialRepository interface {
	CreateLike(like *model.Likes) error
	DeleteLike(userID int64, targetType model.LikeTargetType, targetID int64) error
	CreateLikeWithEffects(like *model.Likes, recipientID int64, notification *model.Notifications) error
	DeleteLikeWithEffects(userID int64, targetType model.LikeTargetType, targetID int64) error
	CreateFavorite(favorite *model.Favorites) error
	DeleteFavorite(userID int64, targetType model.FavoriteTargetType, targetID int64) error
	HasLike(userID int64, targetType model.LikeTargetType, targetID int64) (bool, error)
	HasFavorite(userID int64, targetType model.FavoriteTargetType, targetID int64) (bool, error)
	ListLikedTargetIDs(userID int64, targetType model.LikeTargetType, targetIDs []int64) (map[int64]bool, error)
	ListFavoritedTargetIDs(userID int64, targetType model.FavoriteTargetType, targetIDs []int64) (map[int64]bool, error)
	ResourceExists(id int64) (bool, error)
	TeacherExists(id int64) (bool, error)
	CourseExists(id int64) (bool, error)
	TeacherEvaluationExists(id int64) (bool, error)
	CourseEvaluationExists(id int64) (bool, error)
	TeacherEvaluationReplyExists(id int64) (bool, error)
	CourseEvaluationReplyExists(id int64) (bool, error)
	CommentExists(id int64) (bool, error)
	UpdateResourceLikeCount(resourceID int64, delta int) error
	UpdateCommentLikeCount(commentID int64, delta int) error
	CreateNotification(notification *model.Notifications) error
	GetLikeNotificationRecipient(targetType model.LikeTargetType, targetID int64) (int64, error)
	GetResourceOwnerID(resourceID int64) (int64, error)
}

type socialRepository struct {
	db *gorm.DB
}

func NewSocialRepository(db *gorm.DB) SocialRepository {
	return &socialRepository{db: db}
}

func (r *socialRepository) WithTx(tx *gorm.DB) SocialRepository {
	return &socialRepository{db: tx}
}

func (r *socialRepository) CreateLike(like *model.Likes) error {
	return r.db.Create(like).Error
}

func (r *socialRepository) DeleteLike(userID int64, targetType model.LikeTargetType, targetID int64) error {
	return r.db.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Delete(&model.Likes{}).Error
}

func (r *socialRepository) CreateLikeWithEffects(like *model.Likes, recipientID int64, notification *model.Notifications) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(like).Error; err != nil {
			return err
		}

		switch like.TargetType {
		case model.LikeTargetTypeResource:
			if err := tx.Model(&model.Resources{}).Where("id = ?", like.TargetID).
				Update("like_count", gorm.Expr("GREATEST(like_count + 1, 0)")).Error; err != nil {
				return err
			}
		case model.LikeTargetTypeComment:
			if err := tx.Model(&model.Comments{}).Where("id = ?", like.TargetID).
				Update("like_count", gorm.Expr("GREATEST(like_count + 1, 0)")).Error; err != nil {
				return err
			}
		}

		if recipientID > 0 && notification != nil {
			if err := tx.Create(notification).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *socialRepository) DeleteLikeWithEffects(userID int64, targetType model.LikeTargetType, targetID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
			Delete(&model.Likes{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		switch targetType {
		case model.LikeTargetTypeResource:
			return tx.Model(&model.Resources{}).Where("id = ?", targetID).
				Update("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
		case model.LikeTargetTypeComment:
			return tx.Model(&model.Comments{}).Where("id = ?", targetID).
				Update("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
		default:
			return nil
		}
	})
}

func (r *socialRepository) CreateFavorite(favorite *model.Favorites) error {
	return r.db.Create(favorite).Error
}

func (r *socialRepository) DeleteFavorite(userID int64, targetType model.FavoriteTargetType, targetID int64) error {
	return r.db.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Delete(&model.Favorites{}).Error
}

func (r *socialRepository) HasLike(userID int64, targetType model.LikeTargetType, targetID int64) (bool, error) {
	return recordExists(
		r.db.Table("likes").Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID),
	)
}

func (r *socialRepository) HasFavorite(userID int64, targetType model.FavoriteTargetType, targetID int64) (bool, error) {
	return recordExists(
		r.db.Table("favorites").Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID),
	)
}

func (r *socialRepository) ListLikedTargetIDs(userID int64, targetType model.LikeTargetType, targetIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(targetIDs))
	if userID <= 0 || len(targetIDs) == 0 {
		return result, nil
	}

	var ids []int64
	if err := r.db.Table("likes").
		Where("user_id = ? AND target_type = ? AND target_id IN ?", userID, targetType, targetIDs).
		Pluck("target_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}

func (r *socialRepository) ListFavoritedTargetIDs(userID int64, targetType model.FavoriteTargetType, targetIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(targetIDs))
	if userID <= 0 || len(targetIDs) == 0 {
		return result, nil
	}

	var ids []int64
	if err := r.db.Table("favorites").
		Where("user_id = ? AND target_type = ? AND target_id IN ?", userID, targetType, targetIDs).
		Pluck("target_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}

func (r *socialRepository) ResourceExists(id int64) (bool, error) {
	return recordExists(
		r.db.Table("resources").Where("id = ? AND status = ?", id, model.ResourceStatusApproved),
	)
}

func (r *socialRepository) TeacherExists(id int64) (bool, error) {
	return recordExists(
		r.db.Table("teachers").Where("id = ? AND status = ?", id, "active"),
	)
}

func (r *socialRepository) CourseExists(id int64) (bool, error) {
	return recordExists(
		r.db.Table("courses").Where("id = ? AND status = ?", id, model.CourseStatusActive),
	)
}

func (r *socialRepository) TeacherEvaluationExists(id int64) (bool, error) {
	return recordExists(
		r.db.Table("teacher_evaluations").Where("id = ?", id),
	)
}

func (r *socialRepository) CourseEvaluationExists(id int64) (bool, error) {
	return recordExists(
		r.db.Table("course_evaluations").Where("id = ?", id),
	)
}

func (r *socialRepository) TeacherEvaluationReplyExists(id int64) (bool, error) {
	return recordExists(
		r.db.Table("teacher_evaluation_replies").Where("id = ?", id),
	)
}

func (r *socialRepository) CourseEvaluationReplyExists(id int64) (bool, error) {
	return recordExists(
		r.db.Table("course_evaluation_replies").Where("id = ?", id),
	)
}

func (r *socialRepository) CommentExists(id int64) (bool, error) {
	return recordExists(
		r.db.Table("comments").Where("id = ? AND status = ?", id, model.CommentStatusActive),
	)
}

func (r *socialRepository) UpdateResourceLikeCount(resourceID int64, delta int) error {
	return r.db.Model(&model.Resources{}).Where("id = ?", resourceID).
		Update("like_count", gorm.Expr("GREATEST(like_count + ?, 0)", delta)).Error
}

func (r *socialRepository) UpdateCommentLikeCount(commentID int64, delta int) error {
	return r.db.Model(&model.Comments{}).Where("id = ?", commentID).
		Update("like_count", gorm.Expr("GREATEST(like_count + ?, 0)", delta)).Error
}

func (r *socialRepository) CreateNotification(notification *model.Notifications) error {
	return r.db.Create(notification).Error
}

func (r *socialRepository) GetLikeNotificationRecipient(targetType model.LikeTargetType, targetID int64) (int64, error) {
	type result struct {
		UserID int64
	}
	var item result

	var err error
	switch targetType {
	case model.LikeTargetTypeResource:
		err = r.db.Table("resources").Select("uploader_id AS user_id").Where("id = ?", targetID).Scan(&item).Error
	case model.LikeTargetTypeTeacherEvaluation:
		err = r.db.Table("teacher_evaluations").Select("user_id").Where("id = ?", targetID).Scan(&item).Error
	case model.LikeTargetTypeCourseEvaluation:
		err = r.db.Table("course_evaluations").Select("user_id").Where("id = ?", targetID).Scan(&item).Error
	case model.LikeTargetTypeTeacherReply:
		err = r.db.Table("teacher_evaluation_replies").Select("user_id").Where("id = ?", targetID).Scan(&item).Error
	case model.LikeTargetTypeCourseReply:
		err = r.db.Table("course_evaluation_replies").Select("user_id").Where("id = ?", targetID).Scan(&item).Error
	case model.LikeTargetTypeComment:
		err = r.db.Table("comments").Select("user_id").Where("id = ?", targetID).Scan(&item).Error
	default:
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return item.UserID, nil
}

func (r *socialRepository) GetResourceOwnerID(resourceID int64) (int64, error) {
	type result struct {
		UserID int64
	}
	var item result
	if err := r.db.Table("resources").Select("uploader_id AS user_id").Where("id = ?", resourceID).Scan(&item).Error; err != nil {
		return 0, err
	}
	return item.UserID, nil
}

func recordExists(query *gorm.DB) (bool, error) {
	var marker int
	err := query.Select("1").Limit(1).Scan(&marker).Error
	if err != nil {
		return false, err
	}
	return marker == 1, nil
}

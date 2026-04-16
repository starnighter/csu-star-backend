package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"

	"gorm.io/gorm"
)

const (
	resourceUploadRewardPoints = 2
	evaluationRewardPoints     = 1
)

func rewardUserPointsTx(
	tx *gorm.DB,
	userID int64,
	relatedID int64,
	delta int,
	contributionDelta int,
	pointsType model.PointsType,
	reason string,
	title string,
	content string,
) error {
	var user model.Users
	if err := tx.Select("id", "points").Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	balance := user.Points + delta
	if err := tx.Model(&model.Users{}).
		Where("id = ?", userID).
		Update("points", gorm.Expr("points + ?", delta)).Error; err != nil {
		return err
	}

	if err := tx.Create(&model.PointsRecords{
		UserID:    userID,
		Type:      pointsType,
		Delta:     delta,
		Balance:   balance,
		Reason:    reason,
		RelatedID: relatedID,
	}).Error; err != nil {
		return err
	}

	if err := repo.ApplyUserContributionDeltaTx(tx, userID, contributionDelta); err != nil {
		return err
	}

	return tx.Create(buildReadPointsNotification(
		userID,
		title,
		content,
		relatedID,
	)).Error
}

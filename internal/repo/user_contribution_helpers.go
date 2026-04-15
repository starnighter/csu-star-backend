package repo

import (
	"csu-star-backend/internal/model"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxUserContributionLevel = 100

func userContributionLevelByScore(score int) int {
	if score <= 0 {
		return 1
	}

	level := 1
	remaining := score
	for nextLevel := 2; nextLevel <= maxUserContributionLevel; nextLevel += 1 {
		cost := userContributionUpgradeCost(nextLevel)
		if remaining < cost {
			break
		}
		remaining -= cost
		level = nextLevel
	}
	return level
}

func userContributionUpgradeCost(nextLevel int) int {
	if nextLevel <= 10 {
		return 5
	}
	if nextLevel <= 20 {
		return 10
	}
	// 21-30 => 20, 31-40 => 30, ... 91-100 => 90
	return 20 + ((nextLevel-21)/10)*10
}

func ApplyUserContributionDeltaTx(tx *gorm.DB, userID int64, delta int) error {
	if delta == 0 {
		return nil
	}

	var profile model.UserContributions
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&profile).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		profile = model.UserContributions{
			UserID:       userID,
			Contribution: 0,
			Level:        1,
		}
		if err := tx.Create(&profile).Error; err != nil {
			return err
		}
	}

	newScore := profile.Contribution + delta
	if newScore < 0 {
		newScore = 0
	}
	newLevel := userContributionLevelByScore(newScore)
	if newLevel > maxUserContributionLevel {
		newLevel = maxUserContributionLevel
	}

	if newScore == profile.Contribution && newLevel == profile.Level {
		return nil
	}
	return tx.Model(&model.UserContributions{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"contribution": newScore,
			"level":        newLevel,
		}).Error
}

package model

import "time"

type UserContributions struct {
	UserID            int64     `gorm:"primaryKey;column:user_id" json:"user_id,string"`
	ContributionScore int       `gorm:"column:contribution_score;default:0" json:"contribution_score"`
	Level             int16     `gorm:"column:level;default:1" json:"level"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

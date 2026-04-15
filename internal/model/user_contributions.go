package model

import "time"

// UserContributions stores the accumulated contribution score and derived level.
type UserContributions struct {
	UserID       int64     `gorm:"primary_key;type:bigint" json:"user_id,string"`
	Contribution int       `gorm:"type:integer;not null;default:0" json:"contribution"`
	Level        int       `gorm:"type:integer;not null;default:1" json:"level"`
	CreatedAt    time.Time `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"type:autoUpdateTime" json:"updated_at"`
}

func (UserContributions) TableName() string {
	return "user_contributions"
}

package model

import (
	"csu-star-backend/pkg/utils"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserViolations struct {
	ID                 int64          `gorm:"primary_key;autoIncrement:false" json:"id,string"`
	UserID             int64          `gorm:"type:bigint;not null;index" json:"user_id,string"`
	Scope              string         `gorm:"type:varchar(64);not null;index" json:"scope"`
	TriggerKey         string         `gorm:"type:varchar(255);default:''" json:"trigger_key"`
	Reason             string         `gorm:"type:text;not null" json:"reason"`
	Evidence           datatypes.JSON `gorm:"type:jsonb" json:"evidence"`
	PenaltyLevel       int            `gorm:"type:integer;default:0" json:"penalty_level"`
	BanDurationSeconds int64          `gorm:"type:bigint;default:0" json:"ban_duration_seconds"`
	CreatedAt          time.Time      `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (u *UserViolations) BeforeCreate(tx *gorm.DB) error {
	if u.ID == 0 {
		u.ID = utils.GenerateID()
	}
	return nil
}

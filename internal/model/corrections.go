package model

import (
	"database/sql/driver"
	"errors"
	"time"
)

type CorrectionTargetType string

type CorrectionStatus string

const (
	CorrectionTargetTypeCourse  CorrectionTargetType = "course"
	CorrectionTargetTypeTeacher CorrectionTargetType = "teacher"

	CorrectionStatusPending  CorrectionStatus = "pending"
	CorrectionStatusAccepted CorrectionStatus = "accepted"
	CorrectionStatusRejected CorrectionStatus = "rejected"
)

func (c CorrectionTargetType) Value() (driver.Value, error) {
	return string(c), nil
}

func (c *CorrectionTargetType) Scan(src interface{}) error {
	if src == nil {
		*c = ""
	}
	switch s := src.(type) {
	case []byte:
		*c = CorrectionTargetType(s)
	case string:
		*c = CorrectionTargetType(s)
	default:
		return errors.New("不存在的纠错目标类型")
	}
	return nil
}

func (c CorrectionStatus) Value() (driver.Value, error) {
	return string(c), nil
}

func (c *CorrectionStatus) Scan(src interface{}) error {
	if src == nil {
		*c = ""
	}
	switch s := src.(type) {
	case []byte:
		*c = CorrectionStatus(s)
	case string:
		*c = CorrectionStatus(s)
	default:
		return errors.New("不存在的纠错状态")
	}
	return nil
}

type Corrections struct {
	ID             int64                `gorm:"primary_key" json:"id"`
	UserID         int64                `gorm:"type:bigint;not null" json:"user_id"`
	TargetType     CorrectionTargetType `gorm:"type:correction_target_type;not null" json:"target_type"`
	TargetID       int64                `gorm:"type:bigint;not null" json:"target_id"`
	Field          string               `gorm:"type:varchar(64);not null" json:"field"`
	SuggestedValue string               `gorm:"type:text" json:"suggested_value"`
	Status         CorrectionStatus     `gorm:"type:correction_status" json:"status"`
	ProcessorID    int64                `gorm:"type:bigint" json:"processor_id"`
	ProcessAt      time.Time            `gorm:"type:timestamptz" json:"process_at"`
	ProcessNote    string               `gorm:"type:text" json:"process_note"`
	CreatedAt      time.Time            `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time            `gorm:"type:autoUpdateTime" json:"updated_at"`
}

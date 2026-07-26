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

// CorrectionFieldsByTargetType 是每种纠错目标允许修改的字段白名单。
// 三处必须保持一致，任一处漏改都会导致纠错页显示错误或审核失败：
//  1. service.applyCourseCorrection / applyTeacherCorrection 的 field switch（真正落库的地方）
//  2. repo.correctionCurrentValueSQL 的 CASE WHEN（列表里取「当前值」的地方）
//  3. 管理端 constants/admin-enums.ts 的 correctionField*Dict（显示中文字段名的地方）
//
// 前两处由 repo 的 TestCorrectionCurrentValueSQLCoversAllFields 守卫。
var CorrectionFieldsByTargetType = map[CorrectionTargetType][]string{
	CorrectionTargetTypeCourse:  {"name", "description", "course_type"},
	CorrectionTargetTypeTeacher: {"name", "title", "avatar_url", "department_id", "bio", "tutor_type", "homepage_url"},
}

type Corrections struct {
	ID             int64                `gorm:"primary_key" json:"id,string"`
	UserID         int64                `gorm:"type:bigint;not null" json:"user_id,string"`
	TargetType     CorrectionTargetType `gorm:"type:correction_target_type;not null" json:"target_type"`
	TargetID       int64                `gorm:"type:bigint;not null" json:"target_id,string"`
	Field          string               `gorm:"type:varchar(64);not null" json:"field"`
	SuggestedValue string               `gorm:"type:text" json:"suggested_value"`
	Status         CorrectionStatus     `gorm:"type:correction_status" json:"status"`
	ProcessorID    *int64               `gorm:"type:bigint" json:"processor_id,omitempty,string"`
	ProcessAt      *time.Time           `gorm:"type:timestamptz" json:"process_at,omitempty"`
	ProcessNote    string               `gorm:"type:text" json:"process_note"`
	CreatedAt      time.Time            `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time            `gorm:"type:autoUpdateTime" json:"updated_at"`
}

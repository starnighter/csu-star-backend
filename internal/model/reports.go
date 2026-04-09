package model

import (
	"database/sql/driver"
	"errors"
	"time"
)

type ReportTargetType string

type ReportStatus string

const (
	ReportTargetTypeResource          ReportTargetType = "resource"
	ReportTargetTypeCourse            ReportTargetType = "course"
	ReportTargetTypeTeacherEvaluation ReportTargetType = "teacher_evaluation"
	ReportTargetTypeCourseEvaluation  ReportTargetType = "course_evaluation"
	ReportTargetTypeTeacherReply      ReportTargetType = "teacher_evaluation_reply"
	ReportTargetTypeCourseReply       ReportTargetType = "course_evaluation_reply"
	ReportTargetTypeComment           ReportTargetType = "comment"

	ReportStatusPending   ReportStatus = "pending"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

func (r ReportTargetType) Value() (driver.Value, error) {
	return string(r), nil
}

func (r *ReportTargetType) Scan(src interface{}) error {
	if src == nil {
		*r = ""
	}
	switch s := src.(type) {
	case []byte:
		*r = ReportTargetType(s)
	case string:
		*r = ReportTargetType(s)
	default:
		return errors.New("不存在的报告目标类型")
	}
	return nil
}

func (r ReportStatus) Value() (driver.Value, error) {
	return string(r), nil
}

func (r *ReportStatus) Scan(src interface{}) error {
	if src == nil {
		*r = ""
	}
	switch s := src.(type) {
	case []byte:
		*r = ReportStatus(s)
	case string:
		*r = ReportStatus(s)
	default:
		return errors.New("不存在的报告状态")
	}
	return nil
}

type Reports struct {
	ID          int64            `gorm:"primary_key" json:"id,string"`
	UserID      int64            `gorm:"type:bigint;not null" json:"user_id,string"`
	TargetType  ReportTargetType `gorm:"type:report_target_type;not null" json:"target_type"`
	TargetID    int64            `gorm:"type:bigint;not null" json:"target_id,string"`
	Reason      string           `gorm:"type:text;not null" json:"reason"`
	Description string           `gorm:"type:text" json:"description"`
	Status      ReportStatus     `gorm:"type:report_status" json:"status"`
	ProcessorID *int64           `gorm:"type:bigint" json:"processor_id,omitempty,string"`
	ProcessAt   *time.Time       `gorm:"type:timestamptz" json:"process_at,omitempty"`
	ProcessNote string           `gorm:"type:text" json:"process_note"`
	CreatedAt   time.Time        `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time        `gorm:"type:autoUpdateTime" json:"updated_at"`
}

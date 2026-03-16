package model

import (
	"time"

	"gorm.io/datatypes"
)

type Teachers struct {
	ID                 int64          `gorm:"primary_key" json:"id"`
	Name               string         `gorm:"type:varchar(64);not null" json:"name"`
	Title              string         `gorm:"type:varchar(32)" json:"title"`
	DepartmentID       int16          `gorm:"type:smallint;not null" json:"department_id"`
	AvatarUrl          string         `gorm:"type:varchar(500)" json:"avatar_url"`
	Metadata           datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	AvgTeachingScore   float64        `gorm:"type:numeric(3,2)" json:"avg_teaching_score"`
	AvgGradingScore    float64        `gorm:"type:numeric(3,2)" json:"avg_grading_score"`
	AvgAttendanceScore float64        `gorm:"type:numeric(3,2)" json:"avg_attendance_score"`
	ApprovalRate       float64        `gorm:"type:numeric(5,2)" json:"approval_rate"`
	EvalCount          int64          `gorm:"type:integer;default:0" json:"eval_count"`
	CreatedAt          time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

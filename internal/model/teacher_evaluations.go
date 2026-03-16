package model

import "time"

type TeacherEvaluations struct {
	ID              int64          `gorm:"primary_key" json:"id"`
	UserID          int64          `gorm:"type:bigint;not null" json:"user_id"`
	TeacherID       int64          `gorm:"type:bigint;not null" json:"teacher_id"`
	CourseID        int64          `gorm:"type:bigint;not null" json:"course_id"`
	TeachingScore   int            `gorm:"type:integer;not null" json:"teaching_score"`
	GradingScore    int            `gorm:"type:integer;not null" json:"grading_score"`
	AttendanceScore int            `gorm:"type:integer;not null" json:"attendance_score"`
	Comment         string         `gorm:"type:text" json:"comment"`
	IsAnonymous     bool           `gorm:"type:boolean;default:false" json:"is_anonymous"`
	Status          ResourceStatus `gorm:"type:resource_status" json:"status"`
	CreatedAt       time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

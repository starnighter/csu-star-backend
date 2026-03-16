package model

import "time"

type CourseEvaluations struct {
	ID              int64          `gorm:"primary_key" json:"id"`
	UserID          int64          `gorm:"type:bigint;not null" json:"user_id"`
	CourseID        int64          `gorm:"type:bigint;not null" json:"course_id"`
	WorkloadScore   int            `gorm:"type:integer;not null" json:"workload_score"`
	GainScore       int            `gorm:"type:integer;not null" json:"gain_score"`
	DifficultyScore int            `gorm:"type:integer;not null" json:"difficulty_score"`
	Comment         string         `gorm:"type:text" json:"comment"`
	IsAnonymous     bool           `gorm:"type:boolean;default:false" json:"is_anonymous"`
	Status          ResourceStatus `gorm:"type:resource_status" json:"status"`
	CreatedAt       time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

package model

import (
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/datatypes"
)

type CourseType string

const (
	CourseTypePublic    CourseType = "public"
	CourseTypeNonPublic CourseType = "non_public"
)

func (c CourseType) Value() (driver.Value, error) {
	return string(c), nil
}

func (c *CourseType) Scan(src interface{}) error {
	if src == nil {
		*c = ""
	}
	switch v := src.(type) {
	case []byte:
		*c = CourseType(v)
	case string:
		*c = CourseType(v)
	default:
		return errors.New("不存在的课程类型")
	}
	return nil
}

type Courses struct {
	ID                 int64          `gorm:"primary_key" json:"id"`
	Code               string         `gorm:"type:varchar(32);not null" json:"code"`
	Name               string         `gorm:"type:varchar(128);not null" json:"name"`
	DepartmentID       int16          `gorm:"type:smallint;not null" json:"department_id"`
	Credits            float64        `gorm:"type:numeric(3,1)" json:"credits"`
	CourseType         CourseType     `gorm:"type:course_type" json:"course_type"`
	Description        string         `gorm:"type:text" json:"description"`
	Metadata           datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	AvgWorkloadScore   float64        `gorm:"type:numeric(3,2)" json:"avg_workload_score"`
	AvgGainScore       float64        `gorm:"type:numeric(3,2)" json:"avg_gain_score"`
	AvgDifficultyScore float64        `gorm:"type:numeric(3,2)" json:"avg_diff_score"`
	ResourceCount      int            `gorm:"type:integer" json:"resource_count"`
	EvalCount          int            `gorm:"type:integer" json:"eval_count"`
	HotScore           int            `gorm:"type:integer" json:"hot_score"`
	CreatedAt          time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

type CourseTeachers struct {
	ID        int64     `gorm:"primary_key" json:"id"`
	CourseID  int64     `gorm:"type:bigint;not null" json:"course_id"`
	TeacherID int64     `gorm:"type:bigint;not null" json:"teacher_id"`
	Semester  int64     `gorm:"type:varchar(16)" json:"semester"`
	CreatedAt time.Time `gorm:"type:autoCreateTime" json:"created_at"`
}

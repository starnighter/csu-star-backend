package model

import (
	"csu-star-backend/pkg/utils"
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CourseType string
type CourseStatus string
type CourseTeacherRelationStatus string

const (
	CourseTypePublic    CourseType = "public"
	CourseTypeNonPublic CourseType = "non_public"

	CourseStatusActive  CourseStatus = "active"
	CourseStatusDeleted CourseStatus = "deleted"

	CourseTeacherRelationStatusActive   CourseTeacherRelationStatus = "active"
	CourseTeacherRelationStatusCanceled CourseTeacherRelationStatus = "canceled"
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
	ID                    int64          `gorm:"primary_key" json:"id,string"`
	Code                  string         `gorm:"type:varchar(32)" json:"code"`
	Name                  string         `gorm:"type:varchar(128);not null" json:"name"`
	Credits               float64        `gorm:"type:numeric(3,1)" json:"credits"`
	CourseType            CourseType     `gorm:"type:course_type" json:"course_type"`
	Description           string         `gorm:"type:text" json:"description"`
	Metadata              datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	AvgWorkloadScore      float64        `gorm:"type:numeric(3,2)" json:"avg_workload_score"`
	AvgGainScore          float64        `gorm:"type:numeric(3,2)" json:"avg_gain_score"`
	AvgDifficultyScore    float64        `gorm:"type:numeric(3,2)" json:"avg_diff_score"`
	ResourceCount         int            `gorm:"type:integer" json:"resource_count"`
	DownloadTotal         int            `gorm:"type:integer" json:"download_total"`
	ViewTotal             int            `gorm:"type:integer" json:"view_total"`
	LikeTotal             int            `gorm:"type:integer" json:"like_total"`
	FavoriteCount         int            `gorm:"type:integer" json:"favorite_count"`
	ResourceFavoriteCount int            `gorm:"type:integer" json:"resource_favorite_count"`
	EvalCount             int            `gorm:"type:integer" json:"eval_count"`
	Status                CourseStatus   `gorm:"type:varchar(16);default:'active'" json:"status"`
	CreatedAt             time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

func (c *Courses) BeforeCreate(tx *gorm.DB) error {
	if c.ID == 0 {
		c.ID = utils.GenerateID()
	}
	if c.Status == "" {
		c.Status = CourseStatusActive
	}
	return nil
}

type CourseTeachers struct {
	ID         int64                       `gorm:"primary_key" json:"id,string"`
	CourseID   int64                       `gorm:"type:bigint;not null" json:"course_id,string"`
	TeacherID  int64                       `gorm:"type:bigint;not null" json:"teacher_id,string"`
	Status     CourseTeacherRelationStatus `gorm:"type:varchar(16);default:'active'" json:"status"`
	CanceledAt *time.Time                  `gorm:"type:timestamptz" json:"canceled_at,omitempty"`
	CreatedAt  time.Time                   `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time                   `gorm:"type:autoUpdateTime" json:"updated_at"`
}

func (c *CourseTeachers) BeforeCreate(tx *gorm.DB) error {
	if c.ID == 0 {
		c.ID = utils.GenerateID()
	}
	if c.Status == "" {
		c.Status = CourseTeacherRelationStatusActive
	}
	return nil
}

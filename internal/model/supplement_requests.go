package model

import (
	"csu-star-backend/pkg/utils"
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/gorm"
)

type SupplementRequestStatus string

const (
	SupplementRequestStatusPending  SupplementRequestStatus = "pending"
	SupplementRequestStatusApproved SupplementRequestStatus = "approved"
	SupplementRequestStatusRejected SupplementRequestStatus = "rejected"
)

func (s SupplementRequestStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *SupplementRequestStatus) Scan(src interface{}) error {
	if src == nil {
		*s = ""
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*s = SupplementRequestStatus(v)
	case string:
		*s = SupplementRequestStatus(v)
	default:
		return errors.New("不存在的补录申请状态")
	}
	return nil
}

type TeacherSupplementRequests struct {
	ID                int64                   `gorm:"primary_key;autoIncrement:false" json:"id,string"`
	UserID            int64                   `gorm:"type:bigint;not null" json:"user_id,string"`
	Status            SupplementRequestStatus `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`
	Contact           string                  `gorm:"type:varchar(128);not null" json:"contact"`
	TeacherName       string                  `gorm:"type:varchar(128);not null" json:"teacher_name"`
	DepartmentID      int16                   `gorm:"type:smallint;not null" json:"department_id"`
	Remark            string                  `gorm:"type:text" json:"remark"`
	ReviewedBy        *int64                  `gorm:"type:bigint" json:"reviewed_by,string"`
	ReviewedAt        *time.Time              `gorm:"type:timestamptz" json:"reviewed_at"`
	ReviewNote        string                  `gorm:"type:text" json:"review_note"`
	ApprovedTeacherID *int64                  `gorm:"type:bigint" json:"approved_teacher_id,string"`
	CreatedAt         time.Time               `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time               `gorm:"type:autoUpdateTime" json:"updated_at"`
}

func (t *TeacherSupplementRequests) BeforeCreate(tx *gorm.DB) error {
	if t.ID == 0 {
		t.ID = utils.GenerateID()
	}
	return nil
}

type CourseSupplementRequests struct {
	ID               int64                   `gorm:"primary_key;autoIncrement:false" json:"id,string"`
	UserID           int64                   `gorm:"type:bigint;not null" json:"user_id,string"`
	Status           SupplementRequestStatus `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`
	Contact          string                  `gorm:"type:varchar(128);not null" json:"contact"`
	CourseName       string                  `gorm:"type:varchar(128);not null" json:"course_name"`
	CourseType       CourseType              `gorm:"type:varchar(16);not null" json:"course_type"`
	Remark           string                  `gorm:"type:text" json:"remark"`
	ReviewedBy       *int64                  `gorm:"type:bigint" json:"reviewed_by,string"`
	ReviewedAt       *time.Time              `gorm:"type:timestamptz" json:"reviewed_at"`
	ReviewNote       string                  `gorm:"type:text" json:"review_note"`
	ApprovedCourseID *int64                  `gorm:"type:bigint" json:"approved_course_id,string"`
	CreatedAt        time.Time               `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time               `gorm:"type:autoUpdateTime" json:"updated_at"`
}

func (c *CourseSupplementRequests) BeforeCreate(tx *gorm.DB) error {
	if c.ID == 0 {
		c.ID = utils.GenerateID()
	}
	return nil
}

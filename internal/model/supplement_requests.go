package model

import (
	"csu-star-backend/pkg/utils"
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SupplementRequestType string

type SupplementRequestStatus string

const (
	SupplementRequestTypeTeacher SupplementRequestType = "teacher"
	SupplementRequestTypeCourse  SupplementRequestType = "course"

	SupplementRequestStatusPending  SupplementRequestStatus = "pending"
	SupplementRequestStatusApproved SupplementRequestStatus = "approved"
	SupplementRequestStatusRejected SupplementRequestStatus = "rejected"
)

func (t SupplementRequestType) Value() (driver.Value, error) {
	return string(t), nil
}

func (t *SupplementRequestType) Scan(src interface{}) error {
	if src == nil {
		*t = ""
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*t = SupplementRequestType(v)
	case string:
		*t = SupplementRequestType(v)
	default:
		return errors.New("不存在的补录申请类型")
	}
	return nil
}

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

type SupplementRequests struct {
	ID                  int64                   `gorm:"primary_key;autoIncrement:false" json:"id,string"`
	UserID              int64                   `gorm:"type:bigint;not null" json:"user_id,string"`
	RequestType         SupplementRequestType   `gorm:"type:varchar(16);not null" json:"request_type"`
	Status              SupplementRequestStatus `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`
	Contact             string                  `gorm:"type:varchar(128);not null" json:"contact"`
	TeacherName         string                  `gorm:"type:varchar(128)" json:"teacher_name"`
	DepartmentID        *int16                  `gorm:"type:smallint" json:"department_id"`
	RelatedCourseName   string                  `gorm:"type:varchar(128)" json:"related_course_name"`
	RelatedCourseIDs    datatypes.JSON          `gorm:"type:jsonb;default:'[]'" json:"related_course_ids"`
	RelatedCourseNames  datatypes.JSON          `gorm:"type:jsonb;default:'[]'" json:"related_course_names"`
	RelatedTeacherIDs   datatypes.JSON          `gorm:"type:jsonb;default:'[]'" json:"related_teacher_ids"`
	RelatedTeacherNames datatypes.JSON          `gorm:"type:jsonb;default:'[]'" json:"related_teacher_names"`
	CourseName          string                  `gorm:"type:varchar(128)" json:"course_name"`
	CourseType          string                  `gorm:"type:varchar(16)" json:"course_type"`
	Remark              string                  `gorm:"type:text" json:"remark"`
	ReviewedBy          *int64                  `gorm:"type:bigint" json:"reviewed_by,string"`
	ReviewedAt          *time.Time              `gorm:"type:timestamptz" json:"reviewed_at"`
	ReviewNote          string                  `gorm:"type:text" json:"review_note"`
	ApprovedTargetType  string                  `gorm:"type:varchar(16)" json:"approved_target_type"`
	ApprovedTargetID    *int64                  `gorm:"type:bigint" json:"approved_target_id,string"`
	CreatedAt           time.Time               `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time               `gorm:"type:autoUpdateTime" json:"updated_at"`
}

func (s *SupplementRequests) BeforeCreate(tx *gorm.DB) error {
	if s.ID == 0 {
		s.ID = utils.GenerateID()
	}
	return nil
}

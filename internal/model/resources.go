package model

import (
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/datatypes"
)

type ResourceType string

type ResourceStatus string

const (
	ResourceTypeWord  ResourceType = "word"
	ResourceTypeExcel ResourceType = "excel"
	ResourceTypePPT   ResourceType = "ppt"
	ResourceTypePDF   ResourceType = "pdf"
	ResourceTypeNotes ResourceType = "notes"
	ResourceTypeExam  ResourceType = "exam"
	ResourceTypeLab   ResourceType = "lab"
	ResourceTypeOther ResourceType = "other"

	ResourceStatusDraft    ResourceStatus = "draft"
	ResourceStatusPending  ResourceStatus = "pending"
	ResourceStatusApproved ResourceStatus = "approved"
	ResourceStatusRejected ResourceStatus = "rejected"
)

func (r ResourceType) Value() (driver.Value, error) {
	return string(r), nil
}

func (r *ResourceType) Scan(src interface{}) error {
	if src == nil {
		*r = ""
	}
	switch v := src.(type) {
	case []byte:
		*r = ResourceType(v)
	case string:
		*r = ResourceType(v)
	default:
		return errors.New("不存在的资源类型")
	}
	return nil
}

func (r ResourceStatus) Value() (driver.Value, error) {
	return string(r), nil
}

func (r *ResourceStatus) Scan(src interface{}) error {
	if src == nil {
		*r = ""
	}
	switch v := src.(type) {
	case []byte:
		*r = ResourceStatus(v)
	case string:
		*r = ResourceStatus(v)
	default:
		return errors.New("不存在的资源状态")
	}
	return nil
}

type Resources struct {
	ID            int64          `gorm:"primary_key" json:"id"`
	Title         string         `gorm:"type:varchar(128);not null" json:"title"`
	Description   string         `gorm:"type:text" json:"description"`
	UploaderID    int64          `gorm:"type:bigint;not null" json:"uploader_id"`
	CourseID      int64          `gorm:"type:bigint;not null" json:"course_id"`
	Type          ResourceType   `gorm:"type:resource_type;not null" json:"type"`
	Semester      string         `gorm:"type:varchar(16)" json:"semester"`
	Status        ResourceStatus `gorm:"type:resource_status" json:"status"`
	DownloadCount int            `gorm:"type:integer;default:0" json:"download_count"`
	ViewCount     int            `gorm:"type:integer;default:0" json:"view_count"`
	LikeCount     int            `gorm:"type:integer;default:0" json:"like_count"`
	CommentCount  int            `gorm:"type:integer;default:0" json:"comment_count"`
	ReviewerID    int64          `gorm:"type:bigint" json:"reviewer_id"`
	ReviewAt      time.Time      `gorm:"type:timestamptz" json:"review_at"`
	ReviewReason  string         `gorm:"type:text" json:"review_reason"`
	Metadata      datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt     time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

type ResourceTags struct {
	ID         int64     `gorm:"primary_key" json:"id"`
	ResourceID int64     `gorm:"type:bigint;not null" json:"resource_id"`
	TagID      int64     `gorm:"type:bigint;not null" json:"tag_id"`
	CreatedAt  time.Time `gorm:"type:autoCreateTime" json:"created_at"`
}

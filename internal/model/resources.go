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
	ResourceStatusDeleted  ResourceStatus = "deleted"
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
	ID            int64          `gorm:"primary_key" json:"id,string"`
	Title         string         `gorm:"type:varchar(128);not null" json:"title"`
	Description   string         `gorm:"type:text" json:"description"`
	UploaderID    int64          `gorm:"type:bigint;not null" json:"uploader_id,string"`
	CourseID      int64          `gorm:"type:bigint;not null" json:"course_id,string"`
	Type          ResourceType   `gorm:"type:varchar(64);not null" json:"type"`
	Status        ResourceStatus `gorm:"type:resource_status" json:"status"`
	DownloadCount int            `gorm:"type:integer;default:0" json:"download_count"`
	ViewCount     int            `gorm:"type:integer;default:0" json:"view_count"`
	LikeCount     int            `gorm:"type:integer;default:0" json:"like_count"`
	CommentCount  int            `gorm:"type:integer;default:0" json:"comment_count"`
	Metadata      datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt     time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

type ResourceTags struct {
	ID         int64     `gorm:"primary_key" json:"id,string"`
	ResourceID int64     `gorm:"type:bigint;not null" json:"resource_id,string"`
	TagID      int64     `gorm:"type:bigint;not null" json:"tag_id,string"`
	CreatedAt  time.Time `gorm:"type:autoCreateTime" json:"created_at"`
}

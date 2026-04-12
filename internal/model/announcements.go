package model

import (
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/gorm"
)

type AnnouncementType string

const (
	AnnouncementTypeNotice      AnnouncementType = "notice"
	AnnouncementTypeMaintenance AnnouncementType = "maintenance"
	AnnouncementTypeFeature     AnnouncementType = "feature"
)

func (a AnnouncementType) Value() (driver.Value, error) {
	return string(a), nil
}

func (a *AnnouncementType) Scan(src interface{}) error {
	if src == nil {
		*a = ""
	}
	switch v := src.(type) {
	case []byte:
		*a = AnnouncementType(v)
	case string:
		*a = AnnouncementType(v)
	default:
		return errors.New("不存在的通知类型")
	}
	return nil
}

type Announcements struct {
	ID          int64            `gorm:"primary_key" json:"id,string"`
	Title       string           `gorm:"type:varchar(255);not null" json:"title"`
	Content     string           `gorm:"type:text;not null" json:"content"`
	Type        AnnouncementType `gorm:"type:announcement_type;not null" json:"type"`
	IsPinned    bool             `gorm:"type:boolean" json:"is_pinned"`
	IsPublished bool             `gorm:"type:boolean" json:"is_published"`
	PublishedAt time.Time        `gorm:"type:timestamptz" json:"published_at"`
	ExpiresAt   time.Time        `gorm:"type:timestamptz" json:"expires_at"`
	CreatedAt   time.Time        `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time        `gorm:"type:autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt   `gorm:"index" json:"-"`
}

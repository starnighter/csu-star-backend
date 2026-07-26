package model

import (
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/gorm"
)

type WikiSection string

const (
	WikiSectionCompass WikiSection = "compass"
	WikiSectionMajor   WikiSection = "major"
)

func (s WikiSection) Valid() bool {
	return s == WikiSectionCompass || s == WikiSectionMajor
}

func (s WikiSection) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *WikiSection) Scan(src interface{}) error {
	if src == nil {
		*s = ""
	}
	switch v := src.(type) {
	case []byte:
		*s = WikiSection(v)
	case string:
		*s = WikiSection(v)
	default:
		return errors.New("不存在的 wiki 板块类型")
	}
	return nil
}

// WikiCategories 是 major 板块下的分组(学院)。compass 板块无分组。
type WikiCategories struct {
	ID        int64          `gorm:"primary_key" json:"id,string"`
	Section   WikiSection    `gorm:"type:wiki_section;not null" json:"section"`
	Name      string         `gorm:"type:varchar(64);not null" json:"name"`
	SortOrder int            `gorm:"type:integer" json:"sort_order"`
	CreatedAt time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (WikiCategories) TableName() string { return "wiki_categories" }

// WikiDocuments 的 CategoryID 为 NULL 表示板块直属文档
// (compass 全部文档、major 板块的「简介」)。
type WikiDocuments struct {
	ID          int64          `gorm:"primary_key" json:"id,string"`
	Section     WikiSection    `gorm:"type:wiki_section;not null" json:"section"`
	CategoryID  *int64         `gorm:"type:bigint" json:"category_id,string"`
	Slug        string         `gorm:"type:varchar(128);not null" json:"slug"`
	Title       string         `gorm:"type:varchar(128);not null" json:"title"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	SortOrder   int            `gorm:"type:integer" json:"sort_order"`
	IsPublished bool           `gorm:"type:boolean" json:"is_published"`
	PublishedAt *time.Time     `gorm:"type:timestamptz" json:"published_at"`
	CreatedAt   time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (WikiDocuments) TableName() string { return "wiki_documents" }

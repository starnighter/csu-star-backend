package model

import (
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/gorm"
)

// WikiSection 板块 key（对应 wiki_sections.key）。
// 预置常量仅作代码引用；合法性以注册表为准。
type WikiSection string

const (
	WikiSectionCompass WikiSection = "compass"
	WikiSectionMajor   WikiSection = "major"
)

func (s WikiSection) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *WikiSection) Scan(src interface{}) error {
	if src == nil {
		*s = ""
		return nil
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

// WikiSections 板块注册表：可扩展多板块。
type WikiSections struct {
	Key             string         `gorm:"primary_key;type:varchar(32)" json:"key"`
	Title           string         `gorm:"type:varchar(64);not null" json:"title"`
	SortOrder       int            `gorm:"type:integer" json:"sort_order"`
	AllowCategories bool           `gorm:"type:boolean;not null" json:"allow_categories"`
	CreatedAt       time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (WikiSections) TableName() string { return "wiki_sections" }

// WikiCategories 是某板块下的分组（如 major 下的学院）。
// allow_categories=false 的板块不应有分类。
type WikiCategories struct {
	ID        int64          `gorm:"primary_key" json:"id,string"`
	Section   WikiSection    `gorm:"type:varchar(32);not null" json:"section"`
	Name      string         `gorm:"type:varchar(64);not null" json:"name"`
	SortOrder int            `gorm:"type:integer" json:"sort_order"`
	CreatedAt time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (WikiCategories) TableName() string { return "wiki_categories" }

// WikiDocuments 的 CategoryID 为 NULL 表示板块直属文档。
type WikiDocuments struct {
	ID          int64          `gorm:"primary_key" json:"id,string"`
	Section     WikiSection    `gorm:"type:varchar(32);not null" json:"section"`
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

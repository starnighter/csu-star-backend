package model

import (
	"database/sql/driver"
	"errors"
	"time"
)

type FavoriteTargetType string

const (
	FavoriteTargetTypeResource FavoriteTargetType = "resource"
	FavoriteTargetTypeCourse   FavoriteTargetType = "course"
	FavoriteTargetTypeTeacher  FavoriteTargetType = "teacher"
)

func (f FavoriteTargetType) Value() (driver.Value, error) {
	return string(f), nil
}

func (f *FavoriteTargetType) Scan(src interface{}) error {
	if src == nil {
		*f = ""
	}
	switch v := src.(type) {
	case []byte:
		*f = FavoriteTargetType(v)
	case string:
		*f = FavoriteTargetType(v)
	default:
		return errors.New("不存在的收藏目标类型")
	}
	return nil
}

type Favorites struct {
	ID         int64              `gorm:"primary_key" json:"id"`
	UserID     int64              `gorm:"type:bigint;not null" json:"user_id"`
	TargetType FavoriteTargetType `gorm:"type:favorite_target_type;not null" json:"target_type"`
	TargetID   int64              `gorm:"type:bigint;not null" json:"target_id"`
	CreatedAt  time.Time          `gorm:"type:autoCreateTime" json:"created_at"`
}

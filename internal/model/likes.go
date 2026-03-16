package model

import (
	"database/sql/driver"
	"errors"
	"time"
)

type LikeTargetType string

const (
	LikeTargetTypeResource          LikeTargetType = "resource"
	LikeTargetTypeTeacherEvaluation LikeTargetType = "teacher_evaluation"
	LikeTargetTypeCourseEvaluation  LikeTargetType = "course_evaluation"
	LikeTargetTypeComment           LikeTargetType = "comment"
)

func (l LikeTargetType) Value() (driver.Value, error) {
	return string(l), nil
}

func (l *LikeTargetType) Scan(src interface{}) error {
	if src == nil {
		*l = ""
	}
	switch s := src.(type) {
	case []byte:
		*l = LikeTargetType(s)
	case string:
		*l = LikeTargetType(s)
	default:
		return errors.New("不存在的点赞目标类型")
	}
	return nil
}

type Likes struct {
	ID         int64          `gorm:"primary_key" json:"id"`
	UserID     int64          `gorm:"type:bigint;not null" json:"user_id"`
	TargetType LikeTargetType `gorm:"type:like_target_type;not null" json:"target_type"`
	TargetID   int64          `gorm:"type:bigint;not null" json:"target_id"`
	CreatedAt  time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
}

package model

import (
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/gorm"
)

type CommentStatus string

type CommentTargetType string

const (
	CommentStatusActive  CommentStatus = "active"
	CommentStatusDeleted CommentStatus = "deleted"

	CommentTargetTypeTeacher  CommentTargetType = "teacher"
	CommentTargetTypeCourse   CommentTargetType = "course"
	CommentTargetTypeResource CommentTargetType = "resource"
)

func (c CommentStatus) Value() (driver.Value, error) {
	return string(c), nil
}

func (c *CommentStatus) Scan(src interface{}) error {
	if src == nil {
		*c = ""
	}
	switch s := src.(type) {
	case []byte:
		*c = CommentStatus(s)
	case string:
		*c = CommentStatus(s)
	default:
		return errors.New("不存在的评论状态")
	}
	return nil
}

func (c CommentTargetType) Value() (driver.Value, error) {
	return string(c), nil
}

func (c *CommentTargetType) Scan(src interface{}) error {
	if src == nil {
		*c = ""
	}
	switch s := src.(type) {
	case []byte:
		*c = CommentTargetType(s)
	case string:
		*c = CommentTargetType(s)
	default:
		return errors.New("不存在的评论目标类型")
	}
	return nil
}

type Comments struct {
	ID               int64             `gorm:"primary_key" json:"id,string"`
	TargetType       CommentTargetType `gorm:"type:comment_target_type;not null" json:"target_type"`
	TargetID         int64             `gorm:"type:bigint;not null" json:"target_id,string"`
	UserID           int64             `gorm:"type:bigint;not null" json:"user_id,string"`
	ParentID         *int64            `gorm:"type:bigint" json:"parent_id,string"`
	ReplyToCommentID *int64            `gorm:"type:bigint" json:"reply_to_comment_id,string"`
	Content          string            `gorm:"type:text;not null" json:"content"`
	LikeCount        int               `gorm:"type:integer" json:"like_count"`
	Status           CommentStatus     `gorm:"type:comment_status" json:"status"`
	CreatedAt        time.Time         `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time         `gorm:"type:autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt    `gorm:"type:timestamptz" json:"deleted_at"`
}

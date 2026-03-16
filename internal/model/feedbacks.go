package model

import (
	"time"

	"gorm.io/datatypes"
)

type FeedbackType string

type FeedbackStatus string

const (
	FeedbackTypeBug        FeedbackType = "bug"
	FeedbackTypeSuggestion FeedbackType = "suggestion"
	FeedbackTypeComplaint  FeedbackType = "complaint"
	FeedbackTypeOther      FeedbackType = "other"

	FeedbackStatusPending    FeedbackStatus = "pending"
	FeedbackStatusProcessing FeedbackStatus = "processing"
	FeedbackStatusResolved   FeedbackStatus = "resolved"
	FeedbackStatusClosed     FeedbackStatus = "closed"
)

type Feedbacks struct {
	ID          int64          `gorm:"primary_key" json:"id"`
	UserID      int64          `gorm:"type:bigint;not null" json:"user_id"`
	Type        FeedbackType   `gorm:"type:feedback_type;not null" json:"type"`
	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	Attachments datatypes.JSON `gorm:"type:jsonb" json:"attachments"`
	Status      FeedbackStatus `gorm:"type:feedback_status" json:"status"`
	RepliedBy   int64          `gorm:"type:bigint" json:"replied_by"`
	RepliedAt   time.Time      `gorm:"type:timestamptz" json:"replied_at"`
	Reply       string         `gorm:"type:text" json:"reply"`
	CreatedAt   time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

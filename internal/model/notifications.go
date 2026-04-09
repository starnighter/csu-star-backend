package model

import (
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/datatypes"
)

type NotificationType string

const (
	NotificationTypeSystem            NotificationType = "system"
	NotificationTypeAudit             NotificationType = "audit"
	NotificationTypeLiked             NotificationType = "liked"
	NotificationTypeCommented         NotificationType = "commented"
	NotificationTypeReportHandled     NotificationType = "report_handled"
	NotificationTypeCorrectionHandled NotificationType = "correction_handled"
	NotificationPointsChanged         NotificationType = "points_changed"
)

type NotificationCategory string

const (
	NotificationCategoryAnnouncement NotificationCategory = "announcement"
	NotificationCategoryReport       NotificationCategory = "report"
	NotificationCategoryCorrection   NotificationCategory = "correction"
	NotificationCategoryFeedback     NotificationCategory = "feedback"
	NotificationCategorySupplement   NotificationCategory = "supplement"
	NotificationCategoryAdminMessage NotificationCategory = "admin_message"
	NotificationCategoryPoints       NotificationCategory = "points"
	NotificationCategoryInteraction  NotificationCategory = "interaction"
)

type NotificationResult string

const (
	NotificationResultInform   NotificationResult = "inform"
	NotificationResultApproved NotificationResult = "approved"
	NotificationResultRejected NotificationResult = "rejected"
)

func (n NotificationType) Value() (driver.Value, error) {
	return string(n), nil
}

func (n *NotificationType) Scan(src interface{}) error {
	if src == nil {
		*n = ""
	}
	switch s := src.(type) {
	case []byte:
		*n = NotificationType(s)
	case string:
		*n = NotificationType(s)
	default:
		return errors.New("不存在的通知类型")
	}
	return nil
}

type Notifications struct {
	ID        int64                `gorm:"primary_key" json:"id,string"`
	UserID    int64                `gorm:"type:bigint;not null" json:"user_id,string"`
	Type      NotificationType     `gorm:"type:notification_type;not null" json:"type"`
	Category  NotificationCategory `gorm:"type:varchar(32);not null;default:'admin_message'" json:"category"`
	Result    NotificationResult   `gorm:"type:varchar(16);not null;default:'inform'" json:"result"`
	Title     string               `gorm:"type:varchar(255);not null" json:"title"`
	Content   string               `gorm:"type:text" json:"content"`
	RelatedID int64                `gorm:"type:bigint" json:"related_id,string"`
	IsRead    bool                 `gorm:"type:boolean;default:false" json:"is_read"`
	IsGlobal  bool                 `gorm:"type:boolean;default:false" json:"is_global"`
	Metadata  datatypes.JSON       `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt time.Time            `gorm:"type:autoCreateTime" json:"created_at"`
}

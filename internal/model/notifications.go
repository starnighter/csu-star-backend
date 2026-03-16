package model

import (
	"database/sql/driver"
	"errors"
	"time"
)

type NotificationType string

const (
	NotificationTypeAudit             NotificationType = "audit"
	NotificationTypeLiked             NotificationType = "liked"
	NotificationTypeCommented         NotificationType = "commented"
	NotificationTypeReportHandled     NotificationType = "report_handled"
	NotificationTypeCorrectionHandled NotificationType = "correction_handled"
	NotificationPointsChanged         NotificationType = "points_changed"
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
	ID        int64            `gorm:"primary_key" json:"id"`
	UserID    int64            `gorm:"type:bigint;not null" json:"user_id"`
	Type      NotificationType `gorm:"type:notification_type;not null" json:"type"`
	Title     string           `gorm:"type:varchar(255);not null" json:"title"`
	Content   string           `gorm:"type:text" json:"content"`
	RelatedID int64            `gorm:"type:bigint" json:"related_id"`
	IsRead    bool             `gorm:"type:boolean;default:false" json:"is_read"`
	IsGlobal  bool             `gorm:"type:boolean;default:false" json:"is_global"`
	CreatedAt time.Time        `gorm:"type:autoCreateTime" json:"created_at"`
}

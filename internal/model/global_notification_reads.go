package model

import "time"

type GlobalNotificationReads struct {
	NotificationID int64     `gorm:"primaryKey;type:bigint" json:"notification_id"`
	UserID         int64     `gorm:"primaryKey;type:bigint" json:"user_id"`
	CreatedAt      time.Time `gorm:"type:autoCreateTime" json:"created_at"`
}

package model

import "time"

type SearchHistories struct {
	ID        int64     `gorm:"primary_key" json:"id"`
	UserID    int64     `gorm:"type:bigint;not null" json:"user_id"`
	Keyword   string    `gorm:"type:varchar(255);not null" json:"keyword"`
	CreatedAt time.Time `gorm:"type:autoCreateTime" json:"created_at"`
}

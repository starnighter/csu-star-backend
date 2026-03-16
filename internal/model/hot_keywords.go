package model

import "time"

type HotKeywords struct {
	ID        int64     `gorm:"primary_key" json:"id"`
	Keyword   string    `gorm:"type:varchar(255);not null" json:"keyword"`
	Period    string    `gorm:"type:varchar(16);not null" json:"period"`
	Count     int       `gorm:"type:integer;default:1;not null" json:"count"`
	UpdatedAt time.Time `gorm:"type:autoUpdateTime" json:"updated_at"`
}

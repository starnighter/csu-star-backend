package model

import "time"

type Tags struct {
	ID        int64     `gorm:"primary_key" json:"id"`
	Name      string    `gorm:"type:varchar(64);not null" json:"name"`
	UseCount  int       `gorm:"type:integer;default:0" json:"use_count"`
	CreatedAt time.Time `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:autoUpdateTime" json:"updated_at"`
}

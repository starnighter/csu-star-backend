package model

import "time"

type Departments struct {
	ID        int16     `gorm:"primary_key" json:"id"`
	Name      string    `gorm:"type:varchar(64);not null" json:"name"`
	Code      string    `gorm:"type:varchar(16)" json:"code"`
	CreatedAt time.Time `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:autoUpdateTime" json:"updated_at"`
}

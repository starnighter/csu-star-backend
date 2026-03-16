package model

import "time"

type ResourceFiles struct {
	ID         int64     `gorm:"primary_key" json:"id"`
	ResourceID int64     `gorm:"type:bigint;not null" json:"resource_id"`
	Filename   string    `gorm:"type:varchar(255);not null" json:"filename"`
	FileKey    string    `gorm:"type:varchar(500);not null" json:"file_key"`
	FileSize   int64     `gorm:"type:bigint;not null" json:"file_size"`
	FileHash   string    `gorm:"type:varchar(128)" json:"file_hash"`
	MimeType   string    `gorm:"type:varchar(100)" json:"mime_type"`
	CreatedAt  time.Time `gorm:"type:autoCreateTime" json:"created_at"`
}

package model

import (
	"net"
	"time"
)

type DownloadRecords struct {
	ID         int64     `gorm:"primary_key" json:"id"`
	UserID     int64     `gorm:"type:bigint;not null" json:"user_id"`
	ResourceID int64     `gorm:"type:bigint;not null" json:"resource_id"`
	PointsCost int       `gorm:"type:integer;default:0" json:"points_cost"`
	IpAddress  net.IP    `gorm:"type:inet" json:"ip_address"`
	CreatedAt  time.Time `gorm:"type:autoCreateTime" json:"created_at"`
}

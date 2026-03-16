package model

import (
	"database/sql/driver"
	"errors"
	"time"
)

type PointsType string

const (
	PointsTypeInitial  PointsType = "initial"
	PointsTypeCheckin  PointsType = "checkin"
	PointsTypeUpload   PointsType = "upload"
	PointsTypeDownload PointsType = "download"
	PointsTypeInvite   PointsType = "invite"
	PointsTypeManual   PointsType = "manual"
)

func (p PointsType) Value() (driver.Value, error) {
	return string(p), nil
}

func (p *PointsType) Scan(src interface{}) error {
	if src == nil {
		*p = ""
	}
	switch s := src.(type) {
	case []byte:
		*p = PointsType(s)
	case string:
		*p = PointsType(s)
	default:
		return errors.New("不存在的积分类型")
	}
	return nil
}

type PointsRecords struct {
	ID        int64      `gorm:"primary_key" json:"id"`
	UserID    int64      `gorm:"type:bigint;not null" json:"user_id"`
	Type      PointsType `gorm:"type:points_type;not null" json:"type"`
	Delta     int        `gorm:"type:integer;not null" json:"delta"`
	Balance   int        `gorm:"type:integer;not null" json:"balance"`
	Reason    string     `gorm:"type:text" json:"reason"`
	RelatedID int64      `gorm:"type:bigint" json:"related_id"`
	CreatedAt time.Time  `gorm:"type:autoCreateTime" json:"created_at"`
}

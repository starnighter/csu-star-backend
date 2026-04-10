package model

import (
	"database/sql/driver"
	"errors"
	"net"
	"time"

	"gorm.io/datatypes"
)

type AuditAction string

const (
	AuditActionCreate             AuditAction = "create"
	AuditActionUpdate             AuditAction = "update"
	AuditActionApprove            AuditAction = "approve"
	AuditActionReject             AuditAction = "reject"
	AuditActionDelete             AuditAction = "delete"
	AuditActionBan                AuditAction = "ban"
	AuditActionUnban              AuditAction = "unban"
	AuditActionAutoViolation      AuditAction = "auto_violation"
	AuditActionAutoBan            AuditAction = "auto_ban"
	AuditActionAutoUnban          AuditAction = "auto_unban"
	AuditActionManualAdjustPoints AuditAction = "manual_adjust_points"
)

func (a AuditAction) Value() (driver.Value, error) {
	return string(a), nil
}

func (a *AuditAction) Scan(src interface{}) error {
	if src == nil {
		*a = ""
	}
	switch s := src.(type) {
	case []byte:
		*a = AuditAction(s)
	case string:
		*a = AuditAction(s)
	default:
		return errors.New("不存在的审核动作类型")
	}
	return nil
}

type AuditLogs struct {
	ID         int64          `gorm:"primary_key" json:"id,string"`
	OperatorID int64          `gorm:"type:bigint;not null" json:"operator_id,string"`
	Action     AuditAction    `gorm:"type:audit_action;not null" json:"action"`
	TargetType string         `gorm:"type:varchar(32);not null" json:"target_type"`
	TargetID   int64          `gorm:"type:bigint" json:"target_id,string"`
	OldValues  datatypes.JSON `gorm:"type:jsonb" json:"old_values"`
	NewValues  datatypes.JSON `gorm:"type:jsonb" json:"new_values"`
	Reason     string         `gorm:"type:text" json:"reason"`
	IpAddress  net.IP         `gorm:"type:inet" json:"ip_address"`
	CreatedAt  time.Time      `gorm:"type:timestamp" json:"created_at"`
}

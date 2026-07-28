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
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	// AuditActionApprove / Reject 仅用于「审核」语义：处理举报、采纳或拒绝纠错。
	//
	// 历史上 approve 被复用在了创建公告、编辑课程、回复反馈、建立关联、发送通知等
	// 十余种正向写操作上，导致「approve + course」的真实含义是「管理员编辑了课程」，
	// 审计日志因此无法读懂。现已全部改用 create / update / delete。
	//
	// 注意：audit_action 是 postgres 枚举类型，新增取值需要 ALTER TYPE 迁移，
	// 且枚举值无法删除。若要再细分动作，请先评估迁移与部署顺序（代码先于迁移上线会写库失败）。
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

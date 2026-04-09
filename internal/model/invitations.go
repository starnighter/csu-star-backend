package model

import (
	"database/sql/driver"
	"errors"
	"time"
)

type InvitationStatus string

const (
	InvitationStatusPending InvitationStatus = "pending"
	InvitationStatusInvited InvitationStatus = "invited"
)

func (i InvitationStatus) Value() (driver.Value, error) {
	return string(i), nil
}

func (i *InvitationStatus) Scan(src interface{}) error {
	if src == nil {
		*i = ""
	}
	switch s := src.(type) {
	case []byte:
		*i = InvitationStatus(s)
	case string:
		*i = InvitationStatus(s)
	default:
		return errors.New("不存在的邀请状态")
	}
	return nil
}

type Invitations struct {
	ID        int64            `gorm:"primary_key" json:"id,string"`
	InviterID int64            `gorm:"type:bigint;not null" json:"inviter_id,string"`
	InviteeID *int64           `gorm:"type:bigint" json:"invitee_id,string"`
	Code      string           `gorm:"type:varchar(32);not null" json:"code"`
	Status    InvitationStatus `gorm:"type:invitation_status" json:"status"`
	ExpiresAt *time.Time       `gorm:"type:timestamptz" json:"expires_at"`
	UsedAt    *time.Time       `gorm:"type:timestamptz" json:"used_at"`
	CreatedAt time.Time        `gorm:"type:autoCreateTime" json:"created_at"`
}

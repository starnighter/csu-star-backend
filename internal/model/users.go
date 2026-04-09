package model

import (
	"csu-star-backend/pkg/utils"
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserRole string

type UserStatus string

type OauthProvider string

const (
	UserRoleUser    UserRole = "user"
	UserRoleAdmin   UserRole = "admin"
	UserRoleAuditor UserRole = "auditor"

	UserStatusActive UserStatus = "active"
	UserStatusBanned UserStatus = "banned"

	UserBanSourceAdmin  = "admin"
	UserBanSourceSystem = "system"

	OauthProviderQQ     OauthProvider = "qq"
	OauthProviderWechat OauthProvider = "wechat"
	OauthProviderGithub OauthProvider = "github"
	OauthProviderGoogle OauthProvider = "google"
)

func (r UserRole) Value() (driver.Value, error) {
	return string(r), nil
}

func (r *UserRole) Scan(src interface{}) error {
	if src == nil {
		*r = ""
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*r = UserRole(v)
	case string:
		*r = UserRole(v)
	default:
		return errors.New("不存在的用户角色")
	}
	return nil
}

func (s UserStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *UserStatus) Scan(src interface{}) error {
	if src == nil {
		*s = ""
	}
	switch v := src.(type) {
	case []byte:
		*s = UserStatus(v)
	case string:
		*s = UserStatus(v)
	default:
		return errors.New("不存在的用户状态")
	}
	return nil
}

func (o OauthProvider) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *OauthProvider) Scan(src interface{}) error {
	if src == nil {
		*o = ""
	}
	switch v := src.(type) {
	case []byte:
		*o = OauthProvider(v)
	case string:
		*o = OauthProvider(v)
	default:
		return errors.New("不存在的Oauth提供商")
	}
	return nil
}

type Users struct {
	ID                int64          `gorm:"primary_key;autoIncrement:false" json:"id,string"`
	Email             *string        `gorm:"type:varchar(255);default:null" json:"email"`
	Password          string         `gorm:"type:varchar(255)" json:"password"`
	Nickname          string         `gorm:"type:varchar(64)" json:"nickname"`
	AvatarUrl         string         `gorm:"type:varchar(500)" json:"avatar_url"`
	Role              UserRole       `gorm:"type:user_role;default:'user'" json:"role"`
	Status            UserStatus     `gorm:"type:user_status;default:'active'" json:"status"`
	BanUntil          *time.Time     `gorm:"type:timestamptz" json:"ban_until"`
	BanReason         string         `gorm:"type:varchar(255);default:''" json:"ban_reason"`
	BanSource         string         `gorm:"type:varchar(32);default:''" json:"ban_source"`
	ViolationCount    int            `gorm:"type:integer;default:0" json:"violation_count"`
	LastViolationAt   *time.Time     `gorm:"type:timestamptz" json:"last_violation_at"`
	EmailVerified     bool           `gorm:"type:boolean;default:false" json:"email_verified"`
	Points            int            `gorm:"type:integer;default:5" json:"points"`
	FreeDownloadCount int            `gorm:"type:integer;default:3" json:"free_download_count"`
	InviterID         *int64         `gorm:"type:bigint" json:"inviter_id,string"`
	LastLoginAt       time.Time      `gorm:"type:timestamptz" json:"last_login_at"`
	Metadata          datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt         time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

type UserOauthBinding struct {
	ID        int64          `gorm:"primary_key" json:"id,string"`
	UserID    int64          `gorm:"type:bigint;not null" json:"user_id,string"`
	Provider  OauthProvider  `gorm:"type:oauth_provider;not null" json:"provider"`
	OpenID    string         `gorm:"column:openid;type:varchar(255);not null" json:"open_id"`
	UnionID   string         `gorm:"column:unionid;type:varchar(255)" json:"union_id"`
	BoundAt   time.Time      `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"bound_at"`
	Metadata  datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt time.Time      `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:autoUpdateTime" json:"updated_at"`
}

type UserInfo struct {
	Nickname  string `json:"nickname"`
	AvatarUrl string `json:"avatar_url"`
	OpenID    string `json:"open_id"`
	UnionId   string `json:"union_id"`
}

func (u *Users) BeforeCreate(tx *gorm.DB) error {
	if u.ID == 0 {
		u.ID = utils.GenerateID()
	}
	return nil
}

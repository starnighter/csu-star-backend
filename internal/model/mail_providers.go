package model

import (
	"time"

	"csu-star-backend/pkg/utils"

	"gorm.io/gorm"
)

// MailProviders 是管理端可配置的出站邮件通道。
//
// 它取代了原先只能写在 config-secret.yaml 里的 mail.verification.providers：
// 部署脚本不同步配置文件，改一次发信通道要 ssh 上服务器手动编辑再 SIGHUP，
// 挪到数据库后管理员可以在后台直接维护。config.yaml 里的条目保留为兜底，
// 仅在本表没有任何可用记录时生效。
type MailProviders struct {
	ID int64 `gorm:"primary_key;autoIncrement:false" json:"id,string"`
	// Name 是通道显示名，出现在日志与后台列表里。
	Name string `gorm:"type:varchar(64);not null" json:"name"`
	// Kind: aliyun_dm | tencent_ses | custom_smtp。
	// 云厂商通道恒排在自填 SMTP 之前，与 Tier 无关。
	Kind string `gorm:"type:varchar(32);not null;default:'custom_smtp'" json:"kind"`
	Host string `gorm:"type:varchar(255);not null" json:"host"`
	Port int    `gorm:"type:integer;not null" json:"port"`
	// TLSMode: implicit(465) | starttls(25/587)，空值按 implicit 处理。
	TLSMode  string `gorm:"type:varchar(16);default:'implicit'" json:"tls_mode"`
	Username string `gorm:"type:varchar(255);not null" json:"username"`
	// Password 以 AES-GCM 加密存储（pkg/utils.EncryptSecret），接口永不回传明文。
	Password      string `gorm:"type:text;not null" json:"-"`
	FromEmailAddr string `gorm:"type:varchar(255);not null" json:"from_email_addr"`
	FromName      string `gorm:"type:varchar(64)" json:"from_name"`
	// Tier 只在同一 Kind 内部排序。
	Tier    int  `gorm:"type:integer;default:0" json:"tier"`
	Enabled bool `gorm:"type:boolean;default:true" json:"enabled"`

	// LastOkAt / LastErrAt / LastErr 记录最近一次投递结果，供后台展示健康度。
	LastOkAt  *time.Time `json:"last_ok_at"`
	LastErrAt *time.Time `json:"last_err_at"`
	LastErr   string     `gorm:"type:text" json:"last_err"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MailProviders) TableName() string { return "mail_providers" }

func (p *MailProviders) BeforeCreate(tx *gorm.DB) error {
	if p.ID == 0 {
		p.ID = utils.GenerateID()
	}
	return nil
}

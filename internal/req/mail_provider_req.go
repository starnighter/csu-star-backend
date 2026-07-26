package req

// MailProviderCreateReq 新增邮件通道。
// kind 决定优先级：aliyun_dm / tencent_ses 属于云通道，恒排在 custom_smtp 之前。
type MailProviderCreateReq struct {
	Name          string `json:"name" binding:"required,max=64"`
	Kind          string `json:"kind" binding:"required,oneof=aliyun_dm tencent_ses custom_smtp"`
	Host          string `json:"host" binding:"required,max=255"`
	Port          int    `json:"port" binding:"required,min=1,max=65535"`
	TLSMode       string `json:"tls_mode" binding:"omitempty,oneof=implicit starttls"`
	Username      string `json:"username" binding:"required,max=255"`
	Password      string `json:"password" binding:"required"`
	FromEmailAddr string `json:"from_email_addr" binding:"required,email"`
	FromName      string `json:"from_name" binding:"omitempty,max=64"`
	Tier          int    `json:"tier" binding:"min=0"`
	Enabled       bool   `json:"enabled"`
}

// MailProviderUpdateReq 更新邮件通道。
// password 留空表示不修改，因此列表接口从不需要回传明文密码。
type MailProviderUpdateReq struct {
	Name          string `json:"name" binding:"required,max=64"`
	Kind          string `json:"kind" binding:"required,oneof=aliyun_dm tencent_ses custom_smtp"`
	Host          string `json:"host" binding:"required,max=255"`
	Port          int    `json:"port" binding:"required,min=1,max=65535"`
	TLSMode       string `json:"tls_mode" binding:"omitempty,oneof=implicit starttls"`
	Username      string `json:"username" binding:"required,max=255"`
	Password      string `json:"password"`
	FromEmailAddr string `json:"from_email_addr" binding:"required,email"`
	FromName      string `json:"from_name" binding:"omitempty,max=64"`
	Tier          int    `json:"tier" binding:"min=0"`
	Enabled       bool   `json:"enabled"`
}

type MailProviderTestReq struct {
	To string `json:"to" binding:"required,email"`
}

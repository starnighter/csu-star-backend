package req

type OauthLoginReq struct {
	Provider string `json:"provider" binding:"required,oneof=qq wechat github google"`
	Code     string `json:"code" binding:"required"`
}

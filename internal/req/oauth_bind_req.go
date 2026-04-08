package req

type OauthBindReq struct {
	Provider string       `json:"provider" binding:"required,oneof=qq wechat github google"`
	Code     string       `json:"code" binding:"required"`
	Meta     OauthMetaReq `json:"meta"`
}

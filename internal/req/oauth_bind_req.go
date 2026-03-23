package req

type OauthBindReq struct {
	Provider string `json:"provider" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

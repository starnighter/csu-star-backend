package req

type OauthMetaReq struct {
	CodeVerifier  string `json:"code_verifier"`
	CodeChallenge string `json:"code_challenge"`
}

type OauthLoginReq struct {
	Provider string       `json:"provider" binding:"required,oneof=qq wechat github google"`
	Code     string       `json:"code" binding:"required"`
	Meta     OauthMetaReq `json:"meta"`
}

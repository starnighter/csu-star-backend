package req

type OauthLoginReq struct {
	Provider string         `json:"provider"`
	Code     string         `json:"code"`
	Meta     OauthLoginMeta `json:"meta"`
}

type OauthLoginMeta struct {
	CodeChallenge string `json:"code_challenge"`
}

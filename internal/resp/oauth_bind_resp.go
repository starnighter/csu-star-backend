package resp

type OauthBindResp struct {
	Provider string `json:"provider"`
	BoundAt  string `json:"bound_at"`
}

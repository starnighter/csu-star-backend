package resp

type UserProfileResp struct {
	ID                string `json:"id"`
	Nickname          string `json:"nickname"`
	AvatarUrl         string `json:"avatar_url"`
	Role              string `json:"role"`
	EmailVerified     bool   `json:"email_verified"`
	FreeDownloadCount string `json:"free_download_count"`
}

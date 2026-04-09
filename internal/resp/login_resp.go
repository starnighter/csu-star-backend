package resp

type LoginResp struct {
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    string          `json:"expires_in"`
	UserProfile  UserProfileResp `json:"user_profile"`
}

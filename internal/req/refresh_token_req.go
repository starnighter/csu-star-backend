package req

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

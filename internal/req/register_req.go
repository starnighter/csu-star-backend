package req

type RegisterReq struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	Nickname   string `json:"nickname" binding:"required"`
	AvatarUrl  string `json:"avatar_url" binding:"required"`
	InviteCode string `json:"invite_code" binding:"required"`
}

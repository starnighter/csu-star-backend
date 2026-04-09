package req

type RegisterReq struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8,max=128"`
	Nickname   string `json:"nickname" binding:"omitempty,max=64"`
	AvatarUrl  string `json:"avatar_url" binding:"omitempty"`
	InviteCode string `json:"invite_code" binding:"omitempty,max=64"`
}

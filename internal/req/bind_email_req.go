package req

type BindEmailReq struct {
	Email   string `json:"email" binding:"required,email"`
	Captcha string `json:"captcha" binding:"required,len=6"`
}

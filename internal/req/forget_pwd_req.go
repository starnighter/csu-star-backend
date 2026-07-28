package req

type ForgetPwdReq struct {
	Email    string `json:"email" binding:"required,email"`
	Captcha  string `json:"captcha" binding:"required,len=6"`
	Password string `json:"password" binding:"required"`
}

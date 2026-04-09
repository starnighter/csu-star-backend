package req

type VerifyCaptchaReq struct {
	Email   string `json:"email" binding:"required,email"`
	Captcha string `json:"captcha" binding:"required"`
}

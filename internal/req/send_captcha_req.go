package req

type SendCaptchaReq struct {
	Email string `json:"email" binding:"required,email"`
}

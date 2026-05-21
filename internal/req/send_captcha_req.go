package req

type SendCaptchaReq struct {
	Email   string `json:"email" binding:"required,email"`
	Purpose string `json:"purpose" binding:"required,oneof=register forget_password reset_password bind_email"`
}

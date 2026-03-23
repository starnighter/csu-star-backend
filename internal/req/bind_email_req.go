package req

type BindEmailReq struct {
	Email string `json:"email" binding:"required,email"`
}

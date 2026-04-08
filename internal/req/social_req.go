package req

type LikeReq struct {
	TargetType string `json:"target_type" binding:"required,oneof=resource teacher_evaluation course_evaluation teacher_evaluation_reply course_evaluation_reply comment"`
	TargetID   string `json:"target_id" binding:"required"`
}

type FavoriteReq struct {
	TargetType string `json:"target_type" binding:"required,oneof=resource course teacher"`
	TargetID   string `json:"target_id" binding:"required"`
}

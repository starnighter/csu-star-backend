package req

type LikeReq struct {
	TargetType string `json:"target_type" binding:"required,oneof=resource teacher_evaluation course_evaluation comment"`
	TargetID   int64  `json:"target_id" binding:"required,min=1"`
}

type FavoriteReq struct {
	TargetType string `json:"target_type" binding:"required,oneof=resource course teacher"`
	TargetID   int64  `json:"target_id" binding:"required,min=1"`
}

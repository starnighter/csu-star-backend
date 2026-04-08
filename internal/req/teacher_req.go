package req

type TeacherRankingReq struct {
	RankType     string `form:"rank_type" binding:"omitempty,max=64"`
	DepartmentID int16  `form:"department_id"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	Size         int    `form:"size" binding:"omitempty,min=1,max=100"`
	IsIncreased  bool   `form:"is_increased"`
}

type TeacherSimpleReq struct {
	Q string `form:"q" binding:"omitempty,max=128"`
}

type TeacherEvaluationListReq struct {
	Sort string `form:"sort" binding:"omitempty,oneof=avg_rating likes created_at"`
	Page int    `form:"page" binding:"omitempty,min=1"`
	Size int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type TeacherEvaluationInputReq struct {
	CourseID             string `json:"course_id" binding:"omitempty"`
	RatingQuality        int    `json:"rating_quality" binding:"required,min=1,max=5"`
	RatingGrading        int    `json:"rating_grading" binding:"required,min=1,max=5"`
	RatingAttendance     int    `json:"rating_attendance" binding:"required,min=1,max=5"`
	RatingHomework       *int   `json:"rating_homework" binding:"omitempty,min=1,max=5"`
	RatingGain           *int   `json:"rating_gain" binding:"omitempty,min=1,max=5"`
	RatingExamDifficulty *int   `json:"rating_exam_difficulty" binding:"omitempty,min=1,max=5"`
	Comment              string `json:"comment" binding:"omitempty,max=2000"`
	IsAnonymous          bool   `json:"is_anonymous"`
}

type EvaluationReplyInputReq struct {
	Content        string `json:"content" binding:"required,max=500"`
	ReplyToReplyID string `json:"reply_to_reply_id" binding:"omitempty,min=1"`
	ReplyToUserID  string `json:"reply_to_user_id" binding:"omitempty,min=1"`
	IsAnonymous    bool   `json:"is_anonymous"`
}

type EvaluationReplyUpdateReq struct {
	Content     string `json:"content" binding:"required,max=500"`
	IsAnonymous bool   `json:"is_anonymous"`
}

type MyTeacherEvaluationsReq struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

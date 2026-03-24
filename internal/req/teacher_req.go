package req

type TeacherListReq struct {
	Q            string `form:"q"`
	DepartmentID int16  `form:"department_id"`
	Sort         string `form:"sort" binding:"omitempty,oneof=avg_score avg_quality avg_grading avg_attendance good_rate resource_count eval_count"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	Size         int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type TeacherRankingReq struct {
	RankType     string `form:"rank_type" binding:"omitempty,oneof=avg_score avg_quality avg_grading avg_attendance good_rate resource_count eval_count"`
	Period       string `form:"period" binding:"omitempty,oneof=all month week"`
	DepartmentID int16  `form:"department_id"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	Size         int    `form:"size" binding:"omitempty,min=1,max=100"`
	IsIncreased  bool   `form:"is_increased"`
}

type TeacherEvaluationListReq struct {
	Sort string `form:"sort" binding:"omitempty,oneof=avg_rating likes created_at"`
	Page int    `form:"page" binding:"omitempty,min=1"`
	Size int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type TeacherEvaluationInputReq struct {
	CourseID         int64  `json:"course_id" binding:"required,min=1"`
	RatingQuality    int    `json:"rating_quality" binding:"required,min=1,max=5"`
	RatingGrading    int    `json:"rating_grading" binding:"required,min=1,max=5"`
	RatingAttendance int    `json:"rating_attendance" binding:"required,min=1,max=5"`
	Comment          string `json:"comment" binding:"omitempty,max=2000"`
	IsAnonymous      bool   `json:"is_anonymous"`
}

type MyTeacherEvaluationsReq struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

package req

type CourseListReq struct {
	Q          string `form:"q"`
	CourseType string `form:"course_type" binding:"omitempty,oneof=public non_public required elective experiment"`
	Sort       string `form:"sort" binding:"omitempty,oneof=avg_score avg_homework avg_gain avg_exam_diff resource_count hot hot_score workload_score gain_score difficulty_score"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	Size       int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type CourseRankingReq struct {
	RankType    string `form:"rank_type" binding:"omitempty,oneof=avg_score avg_homework avg_gain avg_exam_diff resource_count hot hot_score"`
	Period      string `form:"period" binding:"omitempty,oneof=all month week"`
	Page        int    `form:"page" binding:"omitempty,min=1"`
	Size        int    `form:"size" binding:"omitempty,min=1,max=100"`
	IsIncreased bool   `form:"is_increased"`
}

type CourseEvaluationListReq struct {
	Sort string `form:"sort" binding:"omitempty,oneof=avg_rating likes created_at"`
	Page int    `form:"page" binding:"omitempty,min=1"`
	Size int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type CourseEvaluationInputReq struct {
	RatingHomework       int    `json:"rating_homework" binding:"required,min=1,max=5"`
	RatingGain           int    `json:"rating_gain" binding:"required,min=1,max=5"`
	RatingExamDifficulty int    `json:"rating_exam_difficulty" binding:"required,min=1,max=5"`
	Comment              string `json:"comment" binding:"omitempty,max=2000"`
	IsAnonymous          bool   `json:"is_anonymous"`
}

type MyCourseEvaluationsReq struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

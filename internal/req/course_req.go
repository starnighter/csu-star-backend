package req

type CourseRankingReq struct {
	RankType    string `form:"rank_type" binding:"omitempty,max=64"`
	Page        int    `form:"page" binding:"omitempty,min=1"`
	Size        int    `form:"size" binding:"omitempty,min=1,max=100"`
	IsIncreased bool   `form:"is_increased"`
}

type CourseEvaluationListReq struct {
	Sort string `form:"sort" binding:"omitempty,oneof=avg_rating likes created_at"`
	Page int    `form:"page" binding:"omitempty,min=1"`
	Size int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type CourseResourceCollectionReq struct {
	Sort         string `form:"sort" binding:"omitempty,oneof=downloads likes created_at"`
	ResourceType string `form:"resource_type" binding:"omitempty"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	Size         int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type CourseEvaluationInputReq struct {
	TeacherID            string `json:"teacher_id" binding:"omitempty"`
	RatingHomework       int    `json:"rating_homework" binding:"required,min=1,max=5"`
	RatingGain           int    `json:"rating_gain" binding:"required,min=1,max=5"`
	RatingExamDifficulty int    `json:"rating_exam_difficulty" binding:"required,min=1,max=5"`
	RatingQuality        *int   `json:"rating_quality" binding:"omitempty,min=1,max=5"`
	RatingGrading        *int   `json:"rating_grading" binding:"omitempty,min=1,max=5"`
	RatingAttendance     *int   `json:"rating_attendance" binding:"omitempty,min=1,max=5"`
	Comment              string `json:"comment" binding:"omitempty,max=2000"`
	IsAnonymous          bool   `json:"is_anonymous"`
}

type ResourceCreateInputReq struct {
	Title        string                      `json:"title" binding:"required,max=128"`
	Description  string                      `json:"description" binding:"omitempty,max=5000"`
	CourseID     string                      `json:"course_id" binding:"required"`
	ResourceType string                      `json:"resource_type" binding:"required,max=64"`
	Files        []ResourceCreateFileItemReq `json:"files" binding:"required,min=1,dive"`
}

type ResourceFinalizeUploadReq struct {
	UploadSessionID string `json:"upload_session_id" binding:"required"`
}

type ResourceCreateFileItemReq struct {
	Filename  string `json:"filename" binding:"required,max=255"`
	SizeBytes int64  `json:"size_bytes" binding:"required,min=1"`
	Mime      string `json:"mime" binding:"omitempty,max=100"`
}

type MyCourseEvaluationsReq struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

type CourseSimpleReq struct {
	Q string `form:"q" binding:"omitempty,max=50"`
}

type CourseTeacherRelationCreateReq struct {
	CourseID  string `json:"course_id" binding:"required"`
	TeacherID string `json:"teacher_id" binding:"required"`
}

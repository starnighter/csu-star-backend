package req

type UpdateMeReq struct {
	Nickname     string `json:"nickname" binding:"omitempty,max=64"`
	AvatarURL    string `json:"avatar_url" binding:"omitempty,max=500"`
	DepartmentID *int16 `json:"department_id" binding:"omitempty,min=1"`
	Grade        *int   `json:"grade" binding:"omitempty,min=2000,max=2100"`
}

type PaginationReq struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

type FavoriteListReq struct {
	TargetType string `form:"target_type" binding:"omitempty,oneof=resource course teacher"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	Size       int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type NotificationListReq struct {
	IsRead *bool `form:"is_read"`
	Page   int   `form:"page" binding:"omitempty,min=1"`
	Size   int   `form:"size" binding:"omitempty,min=1,max=100"`
}

type FeedbackCreateReq struct {
	Type    string   `json:"type" binding:"required,oneof=bug suggestion complaint other"`
	Title   string   `json:"title" binding:"required,max=255"`
	Content string   `json:"content" binding:"required"`
	Files   []string `json:"files"`
}

type ReportCreateReq struct {
	TargetType  string `json:"target_type" binding:"required,oneof=resource course teacher_evaluation course_evaluation teacher_evaluation_reply course_evaluation_reply comment"`
	TargetID    string `json:"target_id" binding:"required"`
	Reason      string `json:"reason" binding:"required"`
	Description string `json:"description"`
}

type CorrectionCreateReq struct {
	TargetType     string `json:"target_type" binding:"required,oneof=course teacher"`
	TargetID       string `json:"target_id" binding:"required"`
	Field          string `json:"field" binding:"required,max=64"`
	SuggestedValue string `json:"suggested_value" binding:"required"`
}

type SearchReq struct {
	Q    string `form:"q" binding:"omitempty,max=50"`
	Type string `form:"type" binding:"omitempty,oneof=all resource course teacher"`
	Page int    `form:"page" binding:"omitempty,min=1"`
	Size int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type SupplementRequestCreateReq struct {
	RequestType       string `json:"request_type" binding:"required,oneof=teacher course"`
	Contact           string `json:"contact" binding:"required,max=128"`
	TeacherName       string `json:"teacher_name" binding:"omitempty,max=128"`
	DepartmentID      *int16 `json:"department_id" binding:"omitempty,min=1"`
	RelatedCourseName string `json:"related_course_name" binding:"omitempty,max=128"`
	CourseName        string `json:"course_name" binding:"omitempty,max=128"`
	CourseType        string `json:"course_type" binding:"omitempty,max=16"`
	Remark            string `json:"remark" binding:"omitempty,max=2000"`
}

type SupplementRequestListReq struct {
	Status      string `form:"status" binding:"omitempty,oneof=pending approved rejected"`
	RequestType string `form:"request_type" binding:"omitempty,oneof=teacher course"`
	Keyword     string `form:"keyword" binding:"omitempty,max=128"`
	Page        int    `form:"page" binding:"omitempty,min=1"`
	Size        int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type SupplementRequestReviewReq struct {
	ReviewNote string `json:"review_note" binding:"omitempty,max=2000"`
}

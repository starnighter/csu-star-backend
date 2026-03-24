package req

type UpdateMeReq struct {
	Nickname  string `json:"nickname" binding:"omitempty,max=64"`
	AvatarURL string `json:"avatar_url" binding:"omitempty,max=500"`
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
	TargetType  string `json:"target_type" binding:"required,oneof=resource teacher_evaluation course_evaluation comment"`
	TargetID    int64  `json:"target_id" binding:"required,min=1"`
	Reason      string `json:"reason" binding:"required"`
	Description string `json:"description"`
}

type CorrectionCreateReq struct {
	TargetType     string `json:"target_type" binding:"required,oneof=course teacher"`
	TargetID       int64  `json:"target_id" binding:"required,min=1"`
	Field          string `json:"field" binding:"required,max=64"`
	SuggestedValue string `json:"suggested_value" binding:"required"`
}

type SearchReq struct {
	Q    string `form:"q" binding:"required"`
	Type string `form:"type" binding:"omitempty,oneof=all resource course teacher"`
	Page int    `form:"page" binding:"omitempty,min=1"`
	Size int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type SearchHotReq struct {
	Period string `form:"period" binding:"omitempty,oneof=day week month"`
}

package req

type AdminPaginationReq struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

type AdminStatisticsReq struct{}

type AdminReportListReq struct {
	Status string `form:"status" binding:"omitempty,oneof=pending resolved dismissed"`
	Page   int    `form:"page" binding:"omitempty,min=1"`
	Size   int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type AdminReportHandleReq struct {
	Action string `json:"action" binding:"required,oneof=resolve dismiss"`
	Remark string `json:"remark" binding:"omitempty,max=2000"`
}

type AdminCorrectionListReq struct {
	Status string `form:"status" binding:"omitempty,oneof=pending accepted rejected"`
	Page   int    `form:"page" binding:"omitempty,min=1"`
	Size   int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type AdminCorrectionHandleReq struct {
	Action string `json:"action" binding:"required,oneof=approve reject"`
	Remark string `json:"remark" binding:"omitempty,max=2000"`
}

type AdminFeedbackListReq struct {
	Status string `form:"status" binding:"omitempty,oneof=pending processing resolved closed"`
	Page   int    `form:"page" binding:"omitempty,min=1"`
	Size   int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type AdminFeedbackReplyReq struct {
	Reply  string `json:"reply" binding:"required,max=4000"`
	Status string `json:"status" binding:"omitempty,oneof=processing resolved closed"`
}

type AdminUserListReq struct {
	Status  string `form:"status" binding:"omitempty,oneof=active banned"`
	Role    string `form:"role" binding:"omitempty,oneof=user auditor admin"`
	Keyword string `form:"keyword" binding:"omitempty,max=128"`
	Page    int    `form:"page" binding:"omitempty,min=1"`
	Size    int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type AdminUserAdjustPointsReq struct {
	Delta  int    `json:"delta" binding:"required"`
	Reason string `json:"reason" binding:"required,max=255"`
}

type AdminResourceDeleteReq struct {
	Reason string `json:"reason" binding:"required,max=255"`
}

type AdminUserNotificationReq struct {
	Title   string `json:"title" binding:"required,max=255"`
	Content string `json:"content" binding:"required,max=4000"`
	Result  string `json:"result" binding:"omitempty,oneof=inform approved rejected"`
}

type AdminAnnouncementListReq struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

type AdminAnnouncementInput struct {
	Title       string  `json:"title" binding:"omitempty,max=255"`
	Content     string  `json:"content" binding:"omitempty"`
	Type        string  `json:"type" binding:"omitempty,oneof=notice maintenance feature"`
	IsPinned    *bool   `json:"is_pinned"`
	IsPublished *bool   `json:"is_published"`
	PublishedAt *string `json:"published_at"`
	ExpiresAt   *string `json:"expires_at"`
}

type AdminCourseListReq struct {
	Status     string `form:"status" binding:"omitempty,oneof=active deleted"`
	CourseType string `form:"course_type" binding:"omitempty,oneof=public non_public"`
	Keyword    string `form:"keyword" binding:"omitempty,max=128"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	Size       int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type AdminCourseInput struct {
	Name        string   `json:"name" binding:"required,max=128"`
	CourseType  string   `json:"course_type" binding:"required,oneof=public non_public"`
	Description string   `json:"description" binding:"omitempty"`
	Credits     *float64 `json:"credits"`
}

type AdminCourseUpdateReq struct {
	Name        string   `json:"name" binding:"omitempty,max=128"`
	CourseType  string   `json:"course_type" binding:"omitempty,oneof=public non_public"`
	Description *string  `json:"description"`
	Credits     *float64 `json:"credits"`
}

type AdminCourseRelationInputReq struct {
	TeacherID string `json:"teacher_id" binding:"required"`
}

type AdminTeacherListReq struct {
	Status       string `form:"status" binding:"omitempty,oneof=active deleted"`
	DepartmentID *int16 `form:"department_id" binding:"omitempty,min=1"`
	Keyword      string `form:"keyword" binding:"omitempty,max=128"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	Size         int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type AdminTeacherInput struct {
	Name         string `json:"name" binding:"required,max=64"`
	DepartmentID int16  `json:"department_id" binding:"required,min=1"`
	Title        string `json:"title" binding:"omitempty,max=32"`
	AvatarURL    string `json:"avatar_url" binding:"omitempty,max=500"`
	Bio          string `json:"bio" binding:"omitempty"`
	TutorType    string `json:"tutor_type" binding:"omitempty,max=128"`
	HomepageURL  string `json:"homepage_url" binding:"omitempty,max=500"`
}

type AdminTeacherUpdateReq struct {
	Name         string  `json:"name" binding:"omitempty,max=64"`
	DepartmentID *int16  `json:"department_id" binding:"omitempty,min=1"`
	Title        *string `json:"title"`
	AvatarURL    *string `json:"avatar_url"`
	Bio          *string `json:"bio"`
	TutorType    *string `json:"tutor_type" binding:"omitempty,max=128"`
	HomepageURL  *string `json:"homepage_url" binding:"omitempty,max=500"`
}

type AdminTeacherRelationInputReq struct {
	CourseID string `json:"course_id" binding:"required"`
}

type AdminResourceListReq struct {
	Status       string `form:"status" binding:"omitempty,oneof=draft pending approved rejected deleted"`
	Keyword      string `form:"keyword" binding:"omitempty,max=128"`
	CourseID     int64  `form:"course_id" binding:"omitempty,min=1"`
	ResourceType string `form:"resource_type" binding:"omitempty,max=64"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	Size         int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type AdminAuditLogListReq struct {
	Action     string `form:"action" binding:"omitempty,max=64"`
	OperatorID int64  `form:"operator_id" binding:"omitempty,min=1"`
	TargetType string `form:"target_type" binding:"omitempty,max=64"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	Size       int    `form:"size" binding:"omitempty,min=1,max=100"`
}

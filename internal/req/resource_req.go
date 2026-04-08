package req

type ResourceDownloadReq struct {
	FileID int64 `form:"file_id" binding:"omitempty,min=1"`
}

type ResourceAbortUploadReq struct {
	UploadSessionID string `json:"upload_session_id" binding:"required"`
}

type ResourceRankingReq struct {
	RankType    string `form:"rank_type" binding:"omitempty,max=64"`
	Page        int    `form:"page" binding:"omitempty,min=1"`
	Size        int    `form:"size" binding:"omitempty,min=1,max=100"`
	IsIncreased bool   `form:"is_increased"`
}

type MyResourcesReq struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

type ResourceUpdateReq struct {
	Title        string `json:"title" binding:"required,max=128"`
	Description  string `json:"description" binding:"omitempty,max=5000"`
	CourseID     string `json:"course_id" binding:"required"`
	ResourceType string `json:"resource_type" binding:"required,max=64"`
}

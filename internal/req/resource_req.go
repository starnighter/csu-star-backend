package req

type ResourceListReq struct {
	Q            string `form:"q"`
	CourseID     int64  `form:"course_id" binding:"omitempty,min=1"`
	ResourceType string `form:"resource_type" binding:"omitempty,oneof=word excel ppt pdf notes exam lab other"`
	Sort         string `form:"sort" binding:"omitempty,oneof=hot_score created_at downloads"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	Size         int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type SubmitResourceReq struct {
}

type ResourceDownloadReq struct {
	FileID int64 `form:"file_id" binding:"omitempty,min=1"`
}

type ResourceRankingReq struct {
	RankType    string `form:"rank_type" binding:"omitempty,oneof=hot_score downloads likes comments views"`
	Page        int    `form:"page" binding:"omitempty,min=1"`
	Size        int    `form:"size" binding:"omitempty,min=1,max=100"`
	IsIncreased bool   `form:"is_increased"`
}

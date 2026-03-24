package req

type CommentListReq struct {
	Sort string `form:"sort" binding:"omitempty,oneof=created_at likes"`
	Page int    `form:"page" binding:"omitempty,min=1"`
	Size int    `form:"size" binding:"omitempty,min=1,max=100"`
}

type CommentCreateReq struct {
	Content          string `json:"content" binding:"required,max=2000"`
	ParentID         int64  `json:"parent_id" binding:"omitempty,min=1"`
	ReplyToCommentID int64  `json:"reply_to_comment_id" binding:"omitempty,min=1"`
}

package req

// id 一律以 string 传输(与 json:"id,string" 的序列化约定对齐),handler 层解析。

type WikiCategoryListReq struct {
	Section string `form:"section" binding:"omitempty,oneof=compass major"`
}

type WikiCategoryInput struct {
	Section   string `json:"section" binding:"omitempty,oneof=compass major"`
	Name      string `json:"name" binding:"required,max=64"`
	SortOrder *int   `json:"sort_order"`
}

type WikiReorderReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

type WikiDocListReq struct {
	Section    string `form:"section" binding:"omitempty,oneof=compass major"`
	CategoryID string `form:"category_id" binding:"omitempty"`
	Keyword    string `form:"keyword" binding:"omitempty,max=128"`
	Status     string `form:"status" binding:"omitempty,oneof=published draft"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	Size       int    `form:"size" binding:"omitempty,min=1,max=200"`
}

type WikiDocCreateReq struct {
	Section    string `json:"section" binding:"required,oneof=compass major"`
	CategoryID string `json:"category_id" binding:"omitempty"`
	Slug       string `json:"slug" binding:"omitempty,max=128"`
	Title      string `json:"title" binding:"required,max=128"`
	Content    string `json:"content" binding:"omitempty"`
	SortOrder  *int   `json:"sort_order"`
}

// WikiDocUpdateReq 指针字段区分「未提交」与「置空」,局部更新。
type WikiDocUpdateReq struct {
	CategoryID *string `json:"category_id"`
	Slug       *string `json:"slug" binding:"omitempty,max=128"`
	Title      *string `json:"title" binding:"omitempty,max=128"`
	Content    *string `json:"content"`
	SortOrder  *int    `json:"sort_order"`
}

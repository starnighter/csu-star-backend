package docengine

import (
	"csu-star-backend/internal/model"
	"time"
)

// FeedItem is a plaza feed card (page or collection).
type FeedItem struct {
	Kind         string    `json:"kind"` // essay | collection | guide | major | course
	ID           int64     `json:"id,string"`
	PageID       int64     `json:"page_id,string"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	OwnerID      int64     `json:"owner_id,string"`
	SpaceKey     string    `json:"space_key"`
	CollectionID *int64    `json:"collection_id,string,omitempty"`
	CourseID     *int64    `json:"course_id,string,omitempty"`
	PublishedAt  time.Time `json:"published_at"`
	HotScore     float64   `json:"hot_score"`
	ViewCount    int64     `json:"view_count"`
	CommentCount int64     `json:"comment_count"`
}

// TreeNode is a directory-tree entry.
type TreeNode struct {
	ID       int64      `json:"id,string"`
	Title    string     `json:"title"`
	ParentID *int64     `json:"parent_id,string,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

// Store is the document engine boundary used by CompassService.
// Memory and GORM implementations are both shipped code paths.
type Store interface {
	// Authors
	IsAuthor(userID int64) (bool, error)
	GrantAuthor(userID int64) error
	CreateAuthorApplication(app *model.CompassAuthorApplication) error
	GetAuthorApplication(id int64) (*model.CompassAuthorApplication, error)
	ListAuthorApplications(status string) ([]model.CompassAuthorApplication, error)
	UpdateAuthorApplication(app *model.CompassAuthorApplication) error
	LatestAuthorApplication(userID int64) (*model.CompassAuthorApplication, error)

	// Pages
	CreatePage(page *model.CompassPage) error
	GetPage(id int64) (*model.CompassPage, error)
	UpdatePage(page *model.CompassPage) error
	ListPagesBySpace(spaceKey string) ([]model.CompassPage, error)
	ListChildPages(parentID int64) ([]model.CompassPage, error)
	AppendHistory(h *model.CompassPageHistory) error
	ListHistory(pageID int64) ([]model.CompassPageHistory, error)

	// Writers / edit requests
	IsPageWriter(pageID, userID int64) (bool, error)
	GrantPageWriter(pageID, userID, grantedBy int64) error
	CreateEditRequest(req *model.CompassEditRequest) error
	GetEditRequest(id int64) (*model.CompassEditRequest, error)
	UpdateEditRequest(req *model.CompassEditRequest) error
	ListEditRequestsForPage(pageID int64, status string) ([]model.CompassEditRequest, error)

	// Collections
	CreateCollection(col *model.CompassCollection) error
	GetCollection(id int64) (*model.CompassCollection, error)
	GetCollectionByRoot(rootPageID int64) (*model.CompassCollection, error)

	// Course roots
	GetCourseRoot(courseID int64) (*model.CompassCourseRoot, error)
	SaveCourseRoot(root *model.CompassCourseRoot) error

	// Feed
	ListFeed(tab, contentType string, limit int) ([]FeedItem, error)

	// Comments
	CreateComment(c *model.CompassComment) error
	ListComments(pageID int64) ([]model.CompassComment, error)
}

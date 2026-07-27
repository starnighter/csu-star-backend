package model

import (
	"time"

	"csu-star-backend/pkg/utils"

	"gorm.io/gorm"
)

// Compass space keys (frontend-facing partitions).
const (
	CompassSpacePlaza   = "plaza"
	CompassSpaceGuides  = "guides"
	CompassSpaceMajors  = "majors"
	CompassSpaceCourses = "courses"
)

const (
	CompassContentEssay      = "essay"
	CompassContentCollection = "collection"
	CompassContentGuide      = "guide"
	CompassContentMajor      = "major"
	CompassContentCourse     = "course"
)

const (
	CompassAppPending  = "pending"
	CompassAppApproved = "approved"
	CompassAppRejected = "rejected"
)

// CompassAuthorApplication is the author identity request queue.
type CompassAuthorApplication struct {
	ID           int64      `gorm:"primaryKey" json:"id,string"`
	UserID       int64      `gorm:"not null;index" json:"user_id,string"`
	Reason       string     `gorm:"type:text;not null" json:"reason"`
	Status       string     `gorm:"type:varchar(16);not null;index" json:"status"`
	ReviewerID   *int64     `gorm:"index" json:"reviewer_id,string,omitempty"`
	ReviewRemark string     `gorm:"type:text" json:"review_remark,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CompassAuthorApplication) TableName() string { return "compass_author_applications" }

func (a *CompassAuthorApplication) BeforeCreate(tx *gorm.DB) error {
	if a.ID == 0 {
		a.ID = utils.GenerateID()
	}
	return nil
}

// CompassAuthor marks users who may create essays/collections.
type CompassAuthor struct {
	UserID    int64     `gorm:"primaryKey" json:"user_id,string"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (CompassAuthor) TableName() string { return "compass_authors" }

// CompassPage is a document node (tree body lives here; engine-backed).
type CompassPage struct {
	ID           int64      `gorm:"primaryKey" json:"id,string"`
	SpaceKey     string     `gorm:"type:varchar(32);not null;index" json:"space_key"`
	ParentID     *int64     `gorm:"index" json:"parent_id,string,omitempty"`
	OwnerID      int64      `gorm:"not null;index" json:"owner_id,string"`
	CollectionID *int64     `gorm:"index" json:"collection_id,string,omitempty"`
	CourseID     *int64     `gorm:"index" json:"course_id,string,omitempty"`
	ContentType  string     `gorm:"type:varchar(32);not null;index" json:"content_type"`
	Title        string     `gorm:"type:varchar(256);not null" json:"title"`
	Body         string     `gorm:"type:text;not null" json:"body"`
	SortOrder    int        `gorm:"not null;default:0" json:"sort_order"`
	ViewCount    int64      `gorm:"not null;default:0" json:"view_count"`
	CommentCount int64      `gorm:"not null;default:0" json:"comment_count"`
	EditCount    int64      `gorm:"not null;default:0" json:"edit_count"`
	FavoriteCount int64     `gorm:"not null;default:0" json:"favorite_count"`
	HotScore     float64    `gorm:"not null;default:0;index" json:"hot_score"`
	PublishedAt  time.Time  `gorm:"index" json:"published_at"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CompassPage) TableName() string { return "compass_pages" }

func (p *CompassPage) BeforeCreate(tx *gorm.DB) error {
	if p.ID == 0 {
		p.ID = utils.GenerateID()
	}
	if p.PublishedAt.IsZero() {
		p.PublishedAt = time.Now()
	}
	return nil
}

// CompassPageHistory stores each completed edit snapshot.
type CompassPageHistory struct {
	ID        int64     `gorm:"primaryKey" json:"id,string"`
	PageID    int64     `gorm:"not null;index" json:"page_id,string"`
	EditorID  int64     `gorm:"not null;index" json:"editor_id,string"`
	Title     string    `gorm:"type:varchar(256);not null" json:"title"`
	Body      string    `gorm:"type:text;not null" json:"body"`
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (CompassPageHistory) TableName() string { return "compass_page_histories" }

func (h *CompassPageHistory) BeforeCreate(tx *gorm.DB) error {
	if h.ID == 0 {
		h.ID = utils.GenerateID()
	}
	return nil
}

// CompassPageWriter grants non-owner write access after edit-request approval.
type CompassPageWriter struct {
	PageID    int64     `gorm:"primaryKey" json:"page_id,string"`
	UserID    int64     `gorm:"primaryKey" json:"user_id,string"`
	GrantedBy int64     `gorm:"not null" json:"granted_by,string"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (CompassPageWriter) TableName() string { return "compass_page_writers" }

// CompassEditRequest is a request to edit someone else's page.
type CompassEditRequest struct {
	ID           int64      `gorm:"primaryKey" json:"id,string"`
	PageID       int64      `gorm:"not null;index" json:"page_id,string"`
	ApplicantID  int64      `gorm:"not null;index" json:"applicant_id,string"`
	Reason       string     `gorm:"type:text;not null" json:"reason"`
	Status       string     `gorm:"type:varchar(16);not null;index" json:"status"`
	ReviewerID   *int64     `gorm:"index" json:"reviewer_id,string,omitempty"`
	ReviewRemark string     `gorm:"type:text" json:"review_remark,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CompassEditRequest) TableName() string { return "compass_edit_requests" }

func (r *CompassEditRequest) BeforeCreate(tx *gorm.DB) error {
	if r.ID == 0 {
		r.ID = utils.GenerateID()
	}
	return nil
}

// CompassCollection is author-owned shelf metadata (root page holds the tree).
type CompassCollection struct {
	ID          int64     `gorm:"primaryKey" json:"id,string"`
	OwnerID     int64     `gorm:"not null;index" json:"owner_id,string"`
	RootPageID  int64     `gorm:"not null;uniqueIndex" json:"root_page_id,string"`
	Title       string    `gorm:"type:varchar(256);not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	CoverURL    string    `gorm:"type:varchar(500)" json:"cover_url"`
	ViewCount   int64     `gorm:"not null;default:0" json:"view_count"`
	HotScore    float64   `gorm:"not null;default:0;index" json:"hot_score"`
	PublishedAt time.Time `gorm:"index" json:"published_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CompassCollection) TableName() string { return "compass_collections" }

func (c *CompassCollection) BeforeCreate(tx *gorm.DB) error {
	if c.ID == 0 {
		c.ID = utils.GenerateID()
	}
	if c.PublishedAt.IsZero() {
		c.PublishedAt = time.Now()
	}
	return nil
}

// CompassCourseRoot maps a course to its co-note root page.
type CompassCourseRoot struct {
	CourseID  int64     `gorm:"primaryKey" json:"course_id,string"`
	PageID    int64     `gorm:"not null;uniqueIndex" json:"page_id,string"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (CompassCourseRoot) TableName() string { return "compass_course_roots" }

// CompassComment is a simple page comment for the workbench side panel.
type CompassComment struct {
	ID        int64     `gorm:"primaryKey" json:"id,string"`
	PageID    int64     `gorm:"not null;index" json:"page_id,string"`
	UserID    int64     `gorm:"not null;index" json:"user_id,string"`
	Body      string    `gorm:"type:text;not null" json:"body"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (CompassComment) TableName() string { return "compass_comments" }

func (c *CompassComment) BeforeCreate(tx *gorm.DB) error {
	if c.ID == 0 {
		c.ID = utils.GenerateID()
	}
	return nil
}

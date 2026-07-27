package docengine

import (
	"csu-star-backend/internal/model"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

// GormStore persists compass documents via Postgres/GORM.
type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

// AutoMigrate creates compass tables.
func (s *GormStore) AutoMigrate() error {
	return s.db.AutoMigrate(
		&model.CompassAuthorApplication{},
		&model.CompassAuthor{},
		&model.CompassPage{},
		&model.CompassPageHistory{},
		&model.CompassPageWriter{},
		&model.CompassEditRequest{},
		&model.CompassCollection{},
		&model.CompassCourseRoot{},
		&model.CompassComment{},
	)
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *GormStore) IsAuthor(userID int64) (bool, error) {
	var n int64
	err := s.db.Model(&model.CompassAuthor{}).Where("user_id = ?", userID).Count(&n).Error
	return n > 0, err
}

func (s *GormStore) GrantAuthor(userID int64) error {
	return s.db.Where(model.CompassAuthor{UserID: userID}).
		Attrs(model.CompassAuthor{UserID: userID}).
		FirstOrCreate(&model.CompassAuthor{UserID: userID}).Error
}

func (s *GormStore) CreateAuthorApplication(app *model.CompassAuthorApplication) error {
	return s.db.Create(app).Error
}

func (s *GormStore) GetAuthorApplication(id int64) (*model.CompassAuthorApplication, error) {
	var app model.CompassAuthorApplication
	if err := s.db.First(&app, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &app, nil
}

func (s *GormStore) ListAuthorApplications(status string) ([]model.CompassAuthorApplication, error) {
	q := s.db.Model(&model.CompassAuthorApplication{}).Order("created_at desc")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var list []model.CompassAuthorApplication
	return list, q.Find(&list).Error
}

func (s *GormStore) UpdateAuthorApplication(app *model.CompassAuthorApplication) error {
	return s.db.Save(app).Error
}

func (s *GormStore) LatestAuthorApplication(userID int64) (*model.CompassAuthorApplication, error) {
	var app model.CompassAuthorApplication
	err := s.db.Where("user_id = ?", userID).Order("created_at desc").First(&app).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &app, nil
}

func (s *GormStore) CreatePage(page *model.CompassPage) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		page.HotScore = computeHot(page.ViewCount, page.CommentCount, page.EditCount, page.FavoriteCount)
		if err := tx.Create(page).Error; err != nil {
			return err
		}
		h := &model.CompassPageHistory{
			PageID: page.ID, EditorID: page.OwnerID, Title: page.Title, Body: page.Body,
		}
		return tx.Create(h).Error
	})
}

func (s *GormStore) GetPage(id int64) (*model.CompassPage, error) {
	var p model.CompassPage
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &p, nil
}

func (s *GormStore) UpdatePage(page *model.CompassPage) error {
	// Never overwrite view_count via full Save — views use IncrementViewCount.
	// Update content fields only so concurrent GetPage view bumps cannot clobber body.
	page.HotScore = computeHot(page.ViewCount, page.CommentCount, page.EditCount, page.FavoriteCount)
	return s.db.Model(&model.CompassPage{}).Where("id = ?", page.ID).Updates(map[string]any{
		"title":         page.Title,
		"body":          page.Body,
		"parent_id":     page.ParentID,
		"collection_id": page.CollectionID,
		"course_id":     page.CourseID,
		"content_type":  page.ContentType,
		"space_key":     page.SpaceKey,
		"sort_order":    page.SortOrder,
		"edit_count":    page.EditCount,
		"hot_score":     page.HotScore,
		"updated_at":    time.Now(),
	}).Error
}

func (s *GormStore) IncrementViewCount(id int64) error {
	return s.db.Model(&model.CompassPage{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

func (s *GormStore) ListPagesBySpace(spaceKey string) ([]model.CompassPage, error) {
	var list []model.CompassPage
	return list, s.db.Where("space_key = ?", spaceKey).Order("sort_order asc, id asc").Find(&list).Error
}

func (s *GormStore) ListChildPages(parentID int64) ([]model.CompassPage, error) {
	var list []model.CompassPage
	return list, s.db.Where("parent_id = ?", parentID).Order("sort_order asc, id asc").Find(&list).Error
}

func (s *GormStore) AppendHistory(h *model.CompassPageHistory) error {
	return s.db.Create(h).Error
}

func (s *GormStore) ListHistory(pageID int64) ([]model.CompassPageHistory, error) {
	var list []model.CompassPageHistory
	return list, s.db.Where("page_id = ?", pageID).Order("created_at desc").Find(&list).Error
}

func (s *GormStore) IsPageWriter(pageID, userID int64) (bool, error) {
	var n int64
	err := s.db.Model(&model.CompassPageWriter{}).
		Where("page_id = ? AND user_id = ?", pageID, userID).Count(&n).Error
	return n > 0, err
}

func (s *GormStore) GrantPageWriter(pageID, userID, grantedBy int64) error {
	w := model.CompassPageWriter{PageID: pageID, UserID: userID, GrantedBy: grantedBy}
	return s.db.Where("page_id = ? AND user_id = ?", pageID, userID).
		Assign(w).FirstOrCreate(&w).Error
}

func (s *GormStore) CreateEditRequest(req *model.CompassEditRequest) error {
	return s.db.Create(req).Error
}

func (s *GormStore) GetEditRequest(id int64) (*model.CompassEditRequest, error) {
	var r model.CompassEditRequest
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &r, nil
}

func (s *GormStore) UpdateEditRequest(req *model.CompassEditRequest) error {
	return s.db.Save(req).Error
}

func (s *GormStore) ListEditRequestsForPage(pageID int64, status string) ([]model.CompassEditRequest, error) {
	q := s.db.Where("page_id = ?", pageID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var list []model.CompassEditRequest
	return list, q.Order("created_at desc").Find(&list).Error
}

func (s *GormStore) CreateCollection(col *model.CompassCollection) error {
	return s.db.Create(col).Error
}

func (s *GormStore) GetCollection(id int64) (*model.CompassCollection, error) {
	var c model.CompassCollection
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &c, nil
}

func (s *GormStore) GetCollectionByRoot(rootPageID int64) (*model.CompassCollection, error) {
	var c model.CompassCollection
	if err := s.db.Where("root_page_id = ?", rootPageID).First(&c).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &c, nil
}

func (s *GormStore) GetCourseRoot(courseID int64) (*model.CompassCourseRoot, error) {
	var r model.CompassCourseRoot
	if err := s.db.First(&r, courseID).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &r, nil
}

func (s *GormStore) SaveCourseRoot(root *model.CompassCourseRoot) error {
	return s.db.Save(root).Error
}

func (s *GormStore) ListFeed(tab, contentType string, limit int) ([]FeedItem, error) {
	if limit <= 0 {
		limit = 20
	}
	order := "published_at desc"
	if tab == "hot" {
		order = "hot_score desc, published_at desc"
	}

	items := make([]FeedItem, 0, limit)

	pageQ := s.db.Model(&model.CompassPage{}).
		Where("content_type <> ?", model.CompassContentCollection)
	if contentType != "" && contentType != "all" && contentType != model.CompassContentCollection {
		pageQ = pageQ.Where("content_type = ?", contentType)
	}
	if contentType == model.CompassContentCollection {
		// only collections below
	} else {
		var pages []model.CompassPage
		if err := pageQ.Order(order).Limit(limit).Find(&pages).Error; err != nil {
			return nil, err
		}
		for _, p := range pages {
			kind := p.ContentType
			if kind == "" {
				kind = model.CompassContentEssay
			}
			items = append(items, FeedItem{
				Kind: kind, ID: p.ID, PageID: p.ID, Title: p.Title,
				Summary: gormSummary(p.Body), OwnerID: p.OwnerID, SpaceKey: p.SpaceKey,
				CollectionID: p.CollectionID, CourseID: p.CourseID,
				PublishedAt: p.PublishedAt, HotScore: p.HotScore,
				ViewCount: p.ViewCount, CommentCount: p.CommentCount,
			})
		}
	}

	if contentType == "" || contentType == "all" || contentType == model.CompassContentCollection {
		var cols []model.CompassCollection
		colOrder := "published_at desc"
		if tab == "hot" {
			colOrder = "hot_score desc, published_at desc"
		}
		if err := s.db.Order(colOrder).Limit(limit).Find(&cols).Error; err != nil {
			return nil, err
		}
		for _, c := range cols {
			items = append(items, FeedItem{
				Kind: model.CompassContentCollection, ID: c.ID, PageID: c.RootPageID,
				Title: c.Title, Summary: c.Description, OwnerID: c.OwnerID,
				SpaceKey: model.CompassSpacePlaza, PublishedAt: c.PublishedAt,
				HotScore: c.HotScore, ViewCount: c.ViewCount,
			})
		}
	}

	// re-sort merged
	if tab == "hot" {
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				if items[j].HotScore > items[i].HotScore ||
					(items[j].HotScore == items[i].HotScore && items[j].PublishedAt.After(items[i].PublishedAt)) {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
	} else {
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				if items[j].PublishedAt.After(items[i].PublishedAt) {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func gormSummary(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if utf8.RuneCountInString(body) <= 120 {
		return body
	}
	runes := []rune(body)
	return string(runes[:120]) + "…"
}

func (s *GormStore) CreateComment(c *model.CompassComment) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return err
		}
		return tx.Model(&model.CompassPage{}).Where("id = ?", c.PageID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})
}

func (s *GormStore) ListComments(pageID int64) ([]model.CompassComment, error) {
	var list []model.CompassComment
	return list, s.db.Where("page_id = ?", pageID).Order("created_at asc").Find(&list).Error
}

// EnsureSeedSpaces is a no-op placeholder for future system pages.
func EnsureSeedSpaces(_ *gorm.DB) error { return nil }

// TouchUpdated is used by migrations that need a clock seam.
func TouchUpdated() time.Time { return time.Now() }

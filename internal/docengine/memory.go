package docengine

import (
	"csu-star-backend/internal/model"
	"errors"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound = errors.New("docengine: not found")
)

// MemoryStore is a process-local document engine (tests + optional in-process mode).
type MemoryStore struct {
	mu       sync.RWMutex
	idSeq    int64
	authors  map[int64]struct{}
	apps     map[int64]*model.CompassAuthorApplication
	pages    map[int64]*model.CompassPage
	history  map[int64][]model.CompassPageHistory
	writers  map[string]struct{} // pageID:userID
	editReqs map[int64]*model.CompassEditRequest
	cols     map[int64]*model.CompassCollection
	courses  map[int64]*model.CompassCourseRoot
	comments map[int64][]model.CompassComment
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		authors:  map[int64]struct{}{},
		apps:     map[int64]*model.CompassAuthorApplication{},
		pages:    map[int64]*model.CompassPage{},
		history:  map[int64][]model.CompassPageHistory{},
		writers:  map[string]struct{}{},
		editReqs: map[int64]*model.CompassEditRequest{},
		cols:     map[int64]*model.CompassCollection{},
		courses:  map[int64]*model.CompassCourseRoot{},
		comments: map[int64][]model.CompassComment{},
	}
}

func (s *MemoryStore) nextID() int64 {
	return atomic.AddInt64(&s.idSeq, 1)
}

func writerKey(pageID, userID int64) string {
	return strings.Join([]string{
		itoa(pageID), itoa(userID),
	}, ":")
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [32]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func clonePage(p *model.CompassPage) *model.CompassPage {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

func (s *MemoryStore) IsAuthor(userID int64) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.authors[userID]
	return ok, nil
}

func (s *MemoryStore) GrantAuthor(userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authors[userID] = struct{}{}
	return nil
}

func (s *MemoryStore) CreateAuthorApplication(app *model.CompassAuthorApplication) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if app.ID == 0 {
		app.ID = s.nextID()
	}
	now := time.Now()
	app.CreatedAt = now
	app.UpdatedAt = now
	cp := *app
	s.apps[app.ID] = &cp
	return nil
}

func (s *MemoryStore) GetAuthorApplication(id int64) (*model.CompassAuthorApplication, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *app
	return &cp, nil
}

func (s *MemoryStore) ListAuthorApplications(status string) ([]model.CompassAuthorApplication, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.CompassAuthorApplication, 0)
	for _, app := range s.apps {
		if status != "" && app.Status != status {
			continue
		}
		out = append(out, *app)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) UpdateAuthorApplication(app *model.CompassAuthorApplication) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[app.ID]; !ok {
		return ErrNotFound
	}
	app.UpdatedAt = time.Now()
	cp := *app
	s.apps[app.ID] = &cp
	return nil
}

func (s *MemoryStore) LatestAuthorApplication(userID int64) (*model.CompassAuthorApplication, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best *model.CompassAuthorApplication
	for _, app := range s.apps {
		if app.UserID != userID {
			continue
		}
		if best == nil || app.CreatedAt.After(best.CreatedAt) {
			cp := *app
			best = &cp
		}
	}
	if best == nil {
		return nil, ErrNotFound
	}
	return best, nil
}

func (s *MemoryStore) CreatePage(page *model.CompassPage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if page.ID == 0 {
		page.ID = s.nextID()
	}
	now := time.Now()
	if page.PublishedAt.IsZero() {
		page.PublishedAt = now
	}
	page.CreatedAt = now
	page.UpdatedAt = now
	page.HotScore = computeHot(page.ViewCount, page.CommentCount, page.EditCount, page.FavoriteCount)
	s.pages[page.ID] = clonePage(page)
	// initial history
	hid := s.nextID()
	s.history[page.ID] = []model.CompassPageHistory{{
		ID: hid, PageID: page.ID, EditorID: page.OwnerID,
		Title: page.Title, Body: page.Body, CreatedAt: now,
	}}
	return nil
}

func (s *MemoryStore) GetPage(id int64) (*model.CompassPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pages[id]
	if !ok {
		return nil, ErrNotFound
	}
	return clonePage(p), nil
}

func (s *MemoryStore) UpdatePage(page *model.CompassPage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.pages[page.ID]
	if !ok {
		return ErrNotFound
	}
	// Preserve counters that are only mutated via dedicated atomic helpers
	// when callers pass a stale snapshot (e.g. concurrent IncrementViewCount).
	page.ViewCount = existing.ViewCount
	if page.CommentCount < existing.CommentCount {
		page.CommentCount = existing.CommentCount
	}
	page.UpdatedAt = time.Now()
	page.HotScore = computeHot(page.ViewCount, page.CommentCount, page.EditCount, page.FavoriteCount)
	s.pages[page.ID] = clonePage(page)
	return nil
}

func (s *MemoryStore) IncrementViewCount(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pages[id]
	if !ok {
		return ErrNotFound
	}
	p.ViewCount++
	p.HotScore = computeHot(p.ViewCount, p.CommentCount, p.EditCount, p.FavoriteCount)
	return nil
}

func (s *MemoryStore) ListPagesBySpace(spaceKey string) ([]model.CompassPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.CompassPage, 0)
	for _, p := range s.pages {
		if p.SpaceKey == spaceKey {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (s *MemoryStore) ListChildPages(parentID int64) ([]model.CompassPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.CompassPage, 0)
	for _, p := range s.pages {
		if p.ParentID != nil && *p.ParentID == parentID {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (s *MemoryStore) AppendHistory(h *model.CompassPageHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h.ID == 0 {
		h.ID = s.nextID()
	}
	h.CreatedAt = time.Now()
	s.history[h.PageID] = append(s.history[h.PageID], *h)
	return nil
}

func (s *MemoryStore) ListHistory(pageID int64) ([]model.CompassPageHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.history[pageID]
	out := make([]model.CompassPageHistory, len(list))
	copy(out, list)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) IsPageWriter(pageID, userID int64) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.writers[writerKey(pageID, userID)]
	return ok, nil
}

func (s *MemoryStore) GrantPageWriter(pageID, userID, grantedBy int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writers[writerKey(pageID, userID)] = struct{}{}
	_ = grantedBy
	return nil
}

func (s *MemoryStore) CreateEditRequest(req *model.CompassEditRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.ID == 0 {
		req.ID = s.nextID()
	}
	now := time.Now()
	req.CreatedAt = now
	req.UpdatedAt = now
	cp := *req
	s.editReqs[req.ID] = &cp
	return nil
}

func (s *MemoryStore) GetEditRequest(id int64) (*model.CompassEditRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.editReqs[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (s *MemoryStore) UpdateEditRequest(req *model.CompassEditRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.editReqs[req.ID]; !ok {
		return ErrNotFound
	}
	req.UpdatedAt = time.Now()
	cp := *req
	s.editReqs[req.ID] = &cp
	return nil
}

func (s *MemoryStore) ListEditRequestsForPage(pageID int64, status string) ([]model.CompassEditRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.CompassEditRequest, 0)
	for _, r := range s.editReqs {
		if r.PageID != pageID {
			continue
		}
		if status != "" && r.Status != status {
			continue
		}
		out = append(out, *r)
	}
	return out, nil
}

func (s *MemoryStore) CreateCollection(col *model.CompassCollection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if col.ID == 0 {
		col.ID = s.nextID()
	}
	now := time.Now()
	if col.PublishedAt.IsZero() {
		col.PublishedAt = now
	}
	col.CreatedAt = now
	col.UpdatedAt = now
	cp := *col
	s.cols[col.ID] = &cp
	return nil
}

func (s *MemoryStore) GetCollection(id int64) (*model.CompassCollection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.cols[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (s *MemoryStore) GetCollectionByRoot(rootPageID int64) (*model.CompassCollection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.cols {
		if c.RootPageID == rootPageID {
			cp := *c
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) GetCourseRoot(courseID int64) (*model.CompassCourseRoot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.courses[courseID]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (s *MemoryStore) SaveCourseRoot(root *model.CompassCourseRoot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if root.CreatedAt.IsZero() {
		root.CreatedAt = time.Now()
	}
	cp := *root
	s.courses[root.CourseID] = &cp
	return nil
}

func summaryOf(body string) string {
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

func computeHot(view, comment, edit, fav int64) float64 {
	return float64(view)*1 + float64(comment)*3 + float64(edit)*2 + float64(fav)*4
}

func (s *MemoryStore) ListFeed(tab, contentType string, limit int) ([]FeedItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	items := make([]FeedItem, 0)

	// pages as essays/guide/major/course content
	for _, p := range s.pages {
		// skip collection root pages (they appear as collection cards)
		if p.ContentType == model.CompassContentCollection {
			continue
		}
		kind := p.ContentType
		if kind == "" {
			kind = model.CompassContentEssay
		}
		if contentType != "" && contentType != "all" && kind != contentType {
			continue
		}
		items = append(items, FeedItem{
			Kind: kind, ID: p.ID, PageID: p.ID, Title: p.Title,
			Summary: summaryOf(p.Body), OwnerID: p.OwnerID, SpaceKey: p.SpaceKey,
			CollectionID: p.CollectionID, CourseID: p.CourseID,
			PublishedAt: p.PublishedAt, HotScore: p.HotScore,
			ViewCount: p.ViewCount, CommentCount: p.CommentCount,
		})
	}

	// collections as cards
	if contentType == "" || contentType == "all" || contentType == model.CompassContentCollection {
		for _, c := range s.cols {
			items = append(items, FeedItem{
				Kind: model.CompassContentCollection, ID: c.ID, PageID: c.RootPageID,
				Title: c.Title, Summary: c.Description, OwnerID: c.OwnerID,
				SpaceKey: model.CompassSpacePlaza, PublishedAt: c.PublishedAt,
				HotScore: c.HotScore, ViewCount: c.ViewCount,
			})
		}
	}

	switch tab {
	case "hot":
		sort.Slice(items, func(i, j int) bool {
			if items[i].HotScore == items[j].HotScore {
				return items[i].PublishedAt.After(items[j].PublishedAt)
			}
			return items[i].HotScore > items[j].HotScore
		})
	default: // recent
		sort.Slice(items, func(i, j int) bool {
			return items[i].PublishedAt.After(items[j].PublishedAt)
		})
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *MemoryStore) CreateComment(c *model.CompassComment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == 0 {
		c.ID = s.nextID()
	}
	c.CreatedAt = time.Now()
	s.comments[c.PageID] = append(s.comments[c.PageID], *c)
	if p, ok := s.pages[c.PageID]; ok {
		p.CommentCount++
		p.HotScore = computeHot(p.ViewCount, p.CommentCount, p.EditCount, p.FavoriteCount)
	}
	return nil
}

func (s *MemoryStore) ListComments(pageID int64) ([]model.CompassComment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.comments[pageID]
	out := make([]model.CompassComment, len(list))
	copy(out, list)
	return out, nil
}

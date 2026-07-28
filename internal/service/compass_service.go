package service

import (
	"csu-star-backend/internal/docengine"
	"csu-star-backend/internal/model"
	"errors"
	"strings"
	"time"
)

var (
	ErrCompassUnauthorized   = errors.New("compass: authentication required")
	ErrCompassForbidden      = errors.New("compass: forbidden")
	ErrCompassNotFound       = errors.New("compass: not found")
	ErrCompassNotAuthor      = errors.New("compass: author identity required")
	ErrCompassInvalidPayload = errors.New("compass: invalid payload")
	ErrCompassConflict       = errors.New("compass: conflict")
	ErrCompassNoWrite        = errors.New("compass: write permission denied")
)

// CompassService implements 指北知识广场 business rules.
type CompassService struct {
	store docengine.Store
}

func NewCompassService(store docengine.Store) *CompassService {
	return &CompassService{store: store}
}

func isReviewerRole(role string) bool {
	return role == string(model.UserRoleAdmin) || role == string(model.UserRoleAuditor)
}

func requireUser(userID int64) error {
	if userID <= 0 {
		return ErrCompassUnauthorized
	}
	return nil
}

// ---------- Author applications ----------

func (s *CompassService) ApplyAuthor(userID int64, reason string) (*model.CompassAuthorApplication, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrCompassInvalidPayload
	}
	ok, err := s.store.IsAuthor(userID)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, ErrCompassConflict
	}
	// block duplicate pending
	if latest, err := s.store.LatestAuthorApplication(userID); err == nil && latest.Status == model.CompassAppPending {
		return nil, ErrCompassConflict
	} else if err != nil && !errors.Is(err, docengine.ErrNotFound) {
		return nil, err
	}
	app := &model.CompassAuthorApplication{
		UserID: userID,
		Reason: reason,
		Status: model.CompassAppPending,
	}
	if err := s.store.CreateAuthorApplication(app); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *CompassService) ReviewAuthorApplication(reviewerID int64, reviewerRole string, appID int64, approve bool, remark string) (*model.CompassAuthorApplication, error) {
	if err := requireUser(reviewerID); err != nil {
		return nil, err
	}
	if !isReviewerRole(reviewerRole) {
		return nil, ErrCompassForbidden
	}
	app, err := s.store.GetAuthorApplication(appID)
	if err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, ErrCompassNotFound
		}
		return nil, err
	}
	if app.Status != model.CompassAppPending {
		return nil, ErrCompassConflict
	}
	now := time.Now()
	app.ReviewerID = &reviewerID
	app.ReviewRemark = strings.TrimSpace(remark)
	app.ReviewedAt = &now
	if approve {
		app.Status = model.CompassAppApproved
		if err := s.store.GrantAuthor(app.UserID); err != nil {
			return nil, err
		}
	} else {
		app.Status = model.CompassAppRejected
	}
	if err := s.store.UpdateAuthorApplication(app); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *CompassService) AuthorStatus(userID int64) (isAuthor bool, latest *model.CompassAuthorApplication, err error) {
	if err := requireUser(userID); err != nil {
		return false, nil, err
	}
	isAuthor, err = s.store.IsAuthor(userID)
	if err != nil {
		return false, nil, err
	}
	latest, err = s.store.LatestAuthorApplication(userID)
	if errors.Is(err, docengine.ErrNotFound) {
		return isAuthor, nil, nil
	}
	return isAuthor, latest, err
}

func (s *CompassService) ListAuthorApplications(reviewerRole, status string) ([]model.CompassAuthorApplication, error) {
	if !isReviewerRole(reviewerRole) {
		return nil, ErrCompassForbidden
	}
	return s.store.ListAuthorApplications(status)
}

func (s *CompassService) requireAuthor(userID int64) error {
	ok, err := s.store.IsAuthor(userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCompassNotAuthor
	}
	return nil
}

// ---------- Read (auth required) ----------

func (s *CompassService) GetFeed(userID int64, tab, contentType string, limit int) ([]docengine.FeedItem, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	tab = strings.TrimSpace(tab)
	if tab == "" {
		tab = "recent"
	}
	if tab != "recent" && tab != "hot" {
		return nil, ErrCompassInvalidPayload
	}
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "all"
	}
	return s.store.ListFeed(tab, contentType, limit)
}

// GetPage returns a page for any authenticated user (read).
// canWrite is true only for owner / page writer / admin|auditor — never required to read.
func (s *CompassService) GetPage(userID int64, role string, pageID int64) (*model.CompassPage, bool, error) {
	if err := requireUser(userID); err != nil {
		return nil, false, err
	}
	page, err := s.store.GetPage(pageID)
	if err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, false, ErrCompassNotFound
		}
		return nil, false, err
	}
	// Atomic view bump only — never full Save (would race with concurrent UpdatePage).
	if err := s.store.IncrementViewCount(pageID); err == nil {
		page.ViewCount++
	}
	canWrite, err := s.CanWrite(userID, role, page)
	if err != nil {
		// Writer lookup failure must not block read access.
		return page, false, nil
	}
	return page, canWrite, nil
}

func (s *CompassService) CanWrite(userID int64, role string, page *model.CompassPage) (bool, error) {
	if page == nil {
		return false, ErrCompassNotFound
	}
	if page.OwnerID == userID {
		return true, nil
	}
	if isReviewerRole(role) {
		return true, nil
	}
	return s.store.IsPageWriter(page.ID, userID)
}

func (s *CompassService) GetTree(userID int64, spaceKey string, rootPageID int64) ([]docengine.TreeNode, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	var pages []model.CompassPage
	var err error
	if rootPageID > 0 {
		// subtree: root + all descendants in same space
		root, e := s.store.GetPage(rootPageID)
		if e != nil {
			if errors.Is(e, docengine.ErrNotFound) {
				return nil, ErrCompassNotFound
			}
			return nil, e
		}
		pages, err = s.collectSubtree(root)
	} else {
		if spaceKey == "" {
			return nil, ErrCompassInvalidPayload
		}
		pages, err = s.store.ListPagesBySpace(spaceKey)
	}
	if err != nil {
		return nil, err
	}
	return buildTree(pages, rootPageID), nil
}

func (s *CompassService) collectSubtree(root *model.CompassPage) ([]model.CompassPage, error) {
	out := []model.CompassPage{*root}
	queue := []int64{root.ID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		children, err := s.store.ListChildPages(id)
		if err != nil {
			return nil, err
		}
		out = append(out, children...)
		for _, c := range children {
			queue = append(queue, c.ID)
		}
	}
	return out, nil
}

func buildTree(pages []model.CompassPage, forceRootID int64) []docengine.TreeNode {
	byParent := map[int64][]model.CompassPage{}
	var roots []model.CompassPage
	pageIDs := map[int64]struct{}{}
	for _, p := range pages {
		pageIDs[p.ID] = struct{}{}
	}
	for _, p := range pages {
		if forceRootID > 0 && p.ID == forceRootID {
			roots = append(roots, p)
			continue
		}
		if p.ParentID == nil {
			if forceRootID == 0 {
				roots = append(roots, p)
			}
			continue
		}
		if _, ok := pageIDs[*p.ParentID]; !ok {
			if forceRootID == 0 {
				roots = append(roots, p)
			}
			continue
		}
		byParent[*p.ParentID] = append(byParent[*p.ParentID], p)
	}
	var walk func(p model.CompassPage) docengine.TreeNode
	walk = func(p model.CompassPage) docengine.TreeNode {
		n := docengine.TreeNode{ID: p.ID, Title: p.Title, ParentID: p.ParentID}
		for _, ch := range byParent[p.ID] {
			n.Children = append(n.Children, walk(ch))
		}
		return n
	}
	out := make([]docengine.TreeNode, 0, len(roots))
	for _, r := range roots {
		out = append(out, walk(r))
	}
	return out
}

func (s *CompassService) ListHistory(userID, pageID int64) ([]model.CompassPageHistory, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	if _, err := s.store.GetPage(pageID); err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, ErrCompassNotFound
		}
		return nil, err
	}
	return s.store.ListHistory(pageID)
}

func (s *CompassService) ListComments(userID, pageID int64) ([]model.CompassComment, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	if _, err := s.store.GetPage(pageID); err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, ErrCompassNotFound
		}
		return nil, err
	}
	return s.store.ListComments(pageID)
}

func (s *CompassService) AddComment(userID, pageID int64, body string) (*model.CompassComment, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrCompassInvalidPayload
	}
	if _, err := s.store.GetPage(pageID); err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, ErrCompassNotFound
		}
		return nil, err
	}
	c := &model.CompassComment{PageID: pageID, UserID: userID, Body: body}
	if err := s.store.CreateComment(c); err != nil {
		return nil, err
	}
	return c, nil
}

// ---------- Create essay / collection ----------

type CreateEssayInput struct {
	Title        string
	Body         string
	CollectionID *int64
	CourseID     *int64
	ParentID     *int64
	SpaceKey     string
	ContentType  string
}

func (s *CompassService) CreateEssay(userID int64, in CreateEssayInput) (*model.CompassPage, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	if err := s.requireAuthor(userID); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, ErrCompassInvalidPayload
	}
	space := in.SpaceKey
	ctype := in.ContentType
	var parentID *int64
	var collectionID *int64
	var courseID *int64

	switch {
	case in.CourseID != nil && *in.CourseID > 0:
		root, err := s.EnsureCourseRoot(userID, *in.CourseID, "课程共笔")
		if err != nil {
			return nil, err
		}
		space = model.CompassSpaceCourses
		ctype = model.CompassContentCourse
		pid := root.ID
		parentID = &pid
		cid := *in.CourseID
		courseID = &cid
	case in.CollectionID != nil && *in.CollectionID > 0:
		col, err := s.store.GetCollection(*in.CollectionID)
		if err != nil {
			if errors.Is(err, docengine.ErrNotFound) {
				return nil, ErrCompassNotFound
			}
			return nil, err
		}
		if col.OwnerID != userID {
			return nil, ErrCompassForbidden
		}
		space = model.CompassSpacePlaza
		ctype = model.CompassContentEssay
		pid := col.RootPageID
		parentID = &pid
		cid := col.ID
		collectionID = &cid
	case in.ParentID != nil && *in.ParentID > 0:
		parent, err := s.store.GetPage(*in.ParentID)
		if err != nil {
			if errors.Is(err, docengine.ErrNotFound) {
				return nil, ErrCompassNotFound
			}
			return nil, err
		}
		// must own parent or be writer
		can, err := s.CanWrite(userID, "", parent)
		if err != nil {
			return nil, err
		}
		if !can {
			return nil, ErrCompassNoWrite
		}
		space = parent.SpaceKey
		if ctype == "" {
			ctype = parent.ContentType
			if ctype == model.CompassContentCollection {
				ctype = model.CompassContentEssay
			}
		}
		parentID = in.ParentID
		collectionID = parent.CollectionID
		courseID = parent.CourseID
	default:
		if space == "" {
			space = model.CompassSpacePlaza
		}
		if ctype == "" {
			ctype = model.CompassContentEssay
		}
	}

	page := &model.CompassPage{
		SpaceKey: space, ParentID: parentID, OwnerID: userID,
		CollectionID: collectionID, CourseID: courseID,
		ContentType: ctype, Title: title, Body: in.Body,
	}
	if err := s.store.CreatePage(page); err != nil {
		return nil, err
	}
	return page, nil
}

type CreateCollectionInput struct {
	Title       string
	Description string
	CoverURL    string
}

func (s *CompassService) CreateCollection(userID int64, in CreateCollectionInput) (*model.CompassCollection, *model.CompassPage, error) {
	if err := requireUser(userID); err != nil {
		return nil, nil, err
	}
	if err := s.requireAuthor(userID); err != nil {
		return nil, nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, nil, ErrCompassInvalidPayload
	}
	root := &model.CompassPage{
		SpaceKey: model.CompassSpacePlaza, OwnerID: userID,
		ContentType: model.CompassContentCollection,
		Title:       title,
		Body:        strings.TrimSpace(in.Description),
	}
	if err := s.store.CreatePage(root); err != nil {
		return nil, nil, err
	}
	col := &model.CompassCollection{
		OwnerID: userID, RootPageID: root.ID, Title: title,
		Description: strings.TrimSpace(in.Description),
		CoverURL:    strings.TrimSpace(in.CoverURL),
	}
	if err := s.store.CreateCollection(col); err != nil {
		return nil, nil, err
	}
	// link root to collection
	cid := col.ID
	root.CollectionID = &cid
	_ = s.store.UpdatePage(root)
	return col, root, nil
}

// ---------- Update page (write permission, live, history) ----------

func (s *CompassService) UpdatePage(userID int64, role string, pageID int64, title, body *string) (*model.CompassPage, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	page, err := s.store.GetPage(pageID)
	if err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, ErrCompassNotFound
		}
		return nil, err
	}
	can, err := s.CanWrite(userID, role, page)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrCompassNoWrite
	}
	if title != nil {
		t := strings.TrimSpace(*title)
		if t == "" {
			return nil, ErrCompassInvalidPayload
		}
		page.Title = t
	}
	if body != nil {
		page.Body = *body
	}
	page.EditCount++
	if err := s.store.UpdatePage(page); err != nil {
		return nil, err
	}
	h := &model.CompassPageHistory{
		PageID: page.ID, EditorID: userID, Title: page.Title, Body: page.Body,
	}
	if err := s.store.AppendHistory(h); err != nil {
		return nil, err
	}
	return page, nil
}

// ---------- Edit requests ----------

func (s *CompassService) RequestEdit(userID, pageID int64, reason string) (*model.CompassEditRequest, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrCompassInvalidPayload
	}
	page, err := s.store.GetPage(pageID)
	if err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, ErrCompassNotFound
		}
		return nil, err
	}
	if page.OwnerID == userID {
		return nil, ErrCompassConflict
	}
	can, err := s.store.IsPageWriter(pageID, userID)
	if err != nil {
		return nil, err
	}
	if can {
		return nil, ErrCompassConflict
	}
	// pending duplicate
	pending, err := s.store.ListEditRequestsForPage(pageID, model.CompassAppPending)
	if err != nil {
		return nil, err
	}
	for _, r := range pending {
		if r.ApplicantID == userID {
			return nil, ErrCompassConflict
		}
	}
	req := &model.CompassEditRequest{
		PageID: pageID, ApplicantID: userID, Reason: reason, Status: model.CompassAppPending,
	}
	if err := s.store.CreateEditRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *CompassService) ReviewEditRequest(reviewerID int64, role string, reqID int64, approve bool, remark string) (*model.CompassEditRequest, error) {
	if err := requireUser(reviewerID); err != nil {
		return nil, err
	}
	req, err := s.store.GetEditRequest(reqID)
	if err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, ErrCompassNotFound
		}
		return nil, err
	}
	if req.Status != model.CompassAppPending {
		return nil, ErrCompassConflict
	}
	page, err := s.store.GetPage(req.PageID)
	if err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, ErrCompassNotFound
		}
		return nil, err
	}
	// owner or reviewer/admin
	if page.OwnerID != reviewerID && !isReviewerRole(role) {
		return nil, ErrCompassForbidden
	}
	now := time.Now()
	req.ReviewerID = &reviewerID
	req.ReviewRemark = strings.TrimSpace(remark)
	req.ReviewedAt = &now
	if approve {
		req.Status = model.CompassAppApproved
		if err := s.store.GrantPageWriter(req.PageID, req.ApplicantID, reviewerID); err != nil {
			return nil, err
		}
	} else {
		req.Status = model.CompassAppRejected
	}
	if err := s.store.UpdateEditRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// ---------- Course co-notes ----------

func (s *CompassService) EnsureCourseRoot(userID, courseID int64, courseTitle string) (*model.CompassPage, error) {
	if err := requireUser(userID); err != nil {
		return nil, err
	}
	if courseID <= 0 {
		return nil, ErrCompassInvalidPayload
	}
	if existing, err := s.store.GetCourseRoot(courseID); err == nil {
		page, e := s.store.GetPage(existing.PageID)
		if e != nil {
			return nil, e
		}
		return page, nil
	} else if !errors.Is(err, docengine.ErrNotFound) {
		return nil, err
	}

	title := strings.TrimSpace(courseTitle)
	if title == "" {
		title = "课程共笔"
	}
	cid := courseID
	page := &model.CompassPage{
		SpaceKey: model.CompassSpaceCourses, OwnerID: userID,
		CourseID: &cid, ContentType: model.CompassContentCourse,
		Title: title, Body: "课程共笔合集根页面。作者可在此树下新建笔记。",
	}
	if err := s.store.CreatePage(page); err != nil {
		return nil, err
	}
	root := &model.CompassCourseRoot{CourseID: courseID, PageID: page.ID}
	if err := s.store.SaveCourseRoot(root); err != nil {
		return nil, err
	}
	return page, nil
}

func (s *CompassService) GetCourseRoot(userID, courseID int64) (*model.CompassPage, error) {
	return s.EnsureCourseRoot(userID, courseID, "课程共笔")
}

func (s *CompassService) GetCollection(userID, collectionID int64) (*model.CompassCollection, *model.CompassPage, error) {
	if err := requireUser(userID); err != nil {
		return nil, nil, err
	}
	col, err := s.store.GetCollection(collectionID)
	if err != nil {
		if errors.Is(err, docengine.ErrNotFound) {
			return nil, nil, ErrCompassNotFound
		}
		return nil, nil, err
	}
	page, err := s.store.GetPage(col.RootPageID)
	if err != nil {
		return nil, nil, err
	}
	return col, page, nil
}

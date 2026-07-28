package service

import (
	"csu-star-backend/internal/docengine"
	"csu-star-backend/internal/model"
	"errors"
	"testing"
)

func newTestCompass() *CompassService {
	return NewCompassService(docengine.NewMemoryStore())
}

func TestUnauthenticatedFeedDenied(t *testing.T) {
	svc := newTestCompass()
	_, err := svc.GetFeed(0, "recent", "all", 10)
	if !errors.Is(err, ErrCompassUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestUnauthenticatedPageBodyDenied(t *testing.T) {
	svc := newTestCompass()
	_, _, err := svc.GetPage(0, string(model.UserRoleUser), 1)
	if !errors.Is(err, ErrCompassUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestAuthorApplyApproveCreateEssayAndCollection(t *testing.T) {
	svc := newTestCompass()
	const userID int64 = 100
	const reviewer int64 = 1

	// non-author cannot create
	_, err := svc.CreateEssay(userID, CreateEssayInput{Title: "x", Body: "y"})
	if !errors.Is(err, ErrCompassNotAuthor) {
		t.Fatalf("expected not author, got %v", err)
	}
	_, _, err = svc.CreateCollection(userID, CreateCollectionInput{Title: "col"})
	if !errors.Is(err, ErrCompassNotAuthor) {
		t.Fatalf("expected not author for collection, got %v", err)
	}

	app, err := svc.ApplyAuthor(userID, "想写选课经验")
	if err != nil {
		t.Fatalf("apply author: %v", err)
	}
	if app.Status != model.CompassAppPending {
		t.Fatalf("status want pending, got %s", app.Status)
	}

	// non-reviewer cannot approve
	_, err = svc.ReviewAuthorApplication(userID, string(model.UserRoleUser), app.ID, true, "")
	if !errors.Is(err, ErrCompassForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}

	app, err = svc.ReviewAuthorApplication(reviewer, string(model.UserRoleAdmin), app.ID, true, "ok")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if app.Status != model.CompassAppApproved {
		t.Fatalf("want approved, got %s", app.Status)
	}

	essay, err := svc.CreateEssay(userID, CreateEssayInput{
		Title: "我的选课随笔", Body: "内容A",
	})
	if err != nil {
		t.Fatalf("create essay: %v", err)
	}
	if essay.Title != "我的选课随笔" || essay.ContentType != model.CompassContentEssay {
		t.Fatalf("unexpected essay: %+v", essay)
	}

	col, root, err := svc.CreateCollection(userID, CreateCollectionInput{
		Title: "大一生存", Description: "合集简介",
	})
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if col.RootPageID != root.ID {
		t.Fatalf("root mismatch")
	}

	// essay under collection
	child, err := svc.CreateEssay(userID, CreateEssayInput{
		Title: "合集内第一篇", Body: "子文", CollectionID: &col.ID,
	})
	if err != nil {
		t.Fatalf("create under collection: %v", err)
	}
	if child.ParentID == nil || *child.ParentID != root.ID {
		t.Fatalf("child parent want root %d, got %v", root.ID, child.ParentID)
	}

	// feed recent contains essay + collection
	feed, err := svc.GetFeed(userID, "recent", "all", 50)
	if err != nil {
		t.Fatalf("feed: %v", err)
	}
	var sawEssay, sawCol bool
	for _, it := range feed {
		if it.Kind == model.CompassContentEssay && it.Title == "我的选课随笔" {
			sawEssay = true
		}
		if it.Kind == model.CompassContentCollection && it.Title == "大一生存" {
			sawCol = true
		}
	}
	if !sawEssay || !sawCol {
		t.Fatalf("feed missing items essay=%v col=%v feed=%+v", sawEssay, sawCol, feed)
	}

	// filter essay only
	feedEssays, err := svc.GetFeed(userID, "recent", model.CompassContentEssay, 50)
	if err != nil {
		t.Fatalf("feed essay: %v", err)
	}
	for _, it := range feedEssays {
		if it.Kind != model.CompassContentEssay {
			t.Fatalf("essay filter leaked kind %s", it.Kind)
		}
	}
}

func TestAuthorRejectBlocksCreate(t *testing.T) {
	svc := newTestCompass()
	const userID int64 = 200
	app, err := svc.ApplyAuthor(userID, "reason")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ReviewAuthorApplication(1, string(model.UserRoleAuditor), app.ID, false, "no")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateEssay(userID, CreateEssayInput{Title: "t", Body: "b"})
	if !errors.Is(err, ErrCompassNotAuthor) {
		t.Fatalf("expected not author after reject, got %v", err)
	}
}

func TestEditRequestApproveAllowsWriteRejectDenies(t *testing.T) {
	svc := newTestCompass()
	const owner int64 = 10
	const other int64 = 20
	const admin int64 = 1

	// bootstrap owner as author
	app, _ := svc.ApplyAuthor(owner, "o")
	_, _ = svc.ReviewAuthorApplication(admin, string(model.UserRoleAdmin), app.ID, true, "")
	page, err := svc.CreateEssay(owner, CreateEssayInput{Title: "owner doc", Body: "v1"})
	if err != nil {
		t.Fatal(err)
	}

	// other cannot write
	_, err = svc.UpdatePage(other, string(model.UserRoleUser), page.ID, nil, strPtr("hack"))
	if !errors.Is(err, ErrCompassNoWrite) {
		t.Fatalf("expected no write, got %v", err)
	}

	req, err := svc.RequestEdit(other, page.ID, "想补充")
	if err != nil {
		t.Fatalf("request edit: %v", err)
	}

	// reject path
	_, err = svc.ReviewEditRequest(owner, string(model.UserRoleUser), req.ID, false, "no")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdatePage(other, string(model.UserRoleUser), page.ID, nil, strPtr("hack2"))
	if !errors.Is(err, ErrCompassNoWrite) {
		t.Fatalf("still denied after reject, got %v", err)
	}

	// new request + approve
	req2, err := svc.RequestEdit(other, page.ID, "再申请")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ReviewEditRequest(owner, string(model.UserRoleUser), req2.ID, true, "ok")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := svc.UpdatePage(other, string(model.UserRoleUser), page.ID, nil, strPtr("v2 by collaborator"))
	if err != nil {
		t.Fatalf("write after approve: %v", err)
	}
	if updated.Body != "v2 by collaborator" {
		t.Fatalf("body not updated: %s", updated.Body)
	}

	// history includes initial + new revision
	hist, err := svc.ListHistory(other, page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) < 2 {
		t.Fatalf("expected >=2 history entries, got %d", len(hist))
	}
	// most recent first
	if hist[0].Body != "v2 by collaborator" || hist[0].EditorID != other {
		t.Fatalf("latest history wrong: %+v", hist[0])
	}
}

func TestHotFeedOrdersByScore(t *testing.T) {
	svc := newTestCompass()
	const author int64 = 30
	app, _ := svc.ApplyAuthor(author, "a")
	_, _ = svc.ReviewAuthorApplication(1, string(model.UserRoleAdmin), app.ID, true, "")

	p1, _ := svc.CreateEssay(author, CreateEssayInput{Title: "cold", Body: "c"})
	p2, _ := svc.CreateEssay(author, CreateEssayInput{Title: "hot", Body: "h"})

	// inflate hot on p2 via views
	page2, can, err := svc.GetPage(author, string(model.UserRoleUser), p2.ID)
	if err != nil || !can {
		t.Fatalf("get p2: %v can=%v", err, can)
	}
	// extra views
	for i := 0; i < 5; i++ {
		_, _, _ = svc.GetPage(author, string(model.UserRoleUser), p2.ID)
	}
	_ = page2
	_ = p1

	feed, err := svc.GetFeed(author, "hot", model.CompassContentEssay, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed) < 2 {
		t.Fatalf("want 2 feed items, got %d", len(feed))
	}
	if feed[0].Title != "hot" {
		t.Fatalf("hot should rank first, got %s then %s", feed[0].Title, feed[1].Title)
	}
	if feed[0].HotScore < feed[1].HotScore {
		t.Fatalf("hot score ordering broken: %v < %v", feed[0].HotScore, feed[1].HotScore)
	}
}

func TestCourseRootLazyCreateAndAuth(t *testing.T) {
	svc := newTestCompass()
	const userID int64 = 40
	const courseID int64 = 999

	_, err := svc.GetCourseRoot(0, courseID)
	if !errors.Is(err, ErrCompassUnauthorized) {
		t.Fatalf("unauth course root: %v", err)
	}

	root1, err := svc.GetCourseRoot(userID, courseID)
	if err != nil {
		t.Fatalf("ensure root: %v", err)
	}
	if root1.CourseID == nil || *root1.CourseID != courseID {
		t.Fatalf("course id missing on root")
	}
	if root1.SpaceKey != model.CompassSpaceCourses {
		t.Fatalf("space want courses, got %s", root1.SpaceKey)
	}

	root2, err := svc.GetCourseRoot(userID, courseID)
	if err != nil {
		t.Fatal(err)
	}
	if root2.ID != root1.ID {
		t.Fatalf("lazy root not stable: %d vs %d", root1.ID, root2.ID)
	}

	// author can add child under course
	app, _ := svc.ApplyAuthor(userID, "a")
	_, _ = svc.ReviewAuthorApplication(1, string(model.UserRoleAdmin), app.ID, true, "")
	child, err := svc.CreateEssay(userID, CreateEssayInput{
		Title: "考点", Body: "note", CourseID: &[]int64{courseID}[0],
	})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != root1.ID {
		t.Fatalf("child should hang under course root")
	}

	tree, err := svc.GetTree(userID, model.CompassSpaceCourses, root1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 1 || tree[0].ID != root1.ID {
		t.Fatalf("tree root wrong: %+v", tree)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Title != "考点" {
		t.Fatalf("tree children wrong: %+v", tree[0].Children)
	}
}

func TestHistoryAfterPermittedWrite(t *testing.T) {
	svc := newTestCompass()
	const uid int64 = 55
	app, _ := svc.ApplyAuthor(uid, "a")
	_, _ = svc.ReviewAuthorApplication(1, string(model.UserRoleAdmin), app.ID, true, "")
	page, _ := svc.CreateEssay(uid, CreateEssayInput{Title: "t", Body: "b0"})

	hist0, _ := svc.ListHistory(uid, page.ID)
	n0 := len(hist0)

	_, err := svc.UpdatePage(uid, string(model.UserRoleUser), page.ID, strPtr("t2"), strPtr("b1"))
	if err != nil {
		t.Fatal(err)
	}
	hist1, err := svc.ListHistory(uid, page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist1) != n0+1 {
		t.Fatalf("history len want %d got %d", n0+1, len(hist1))
	}
	if hist1[0].Body != "b1" || hist1[0].EditorID != uid {
		t.Fatalf("history entry: %+v", hist1[0])
	}
}

func strPtr(s string) *string { return &s }

// Concurrent GetPage view bumps must not overwrite body written by UpdatePage.
func TestGetPageViewBumpDoesNotClobberBodyUnderConcurrentWrites(t *testing.T) {
	svc := newTestCompass()
	const uid int64 = 77
	app, _ := svc.ApplyAuthor(uid, "a")
	_, _ = svc.ReviewAuthorApplication(1, string(model.UserRoleAdmin), app.ID, true, "")
	page, err := svc.CreateEssay(uid, CreateEssayInput{Title: "race", Body: "written-0"})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	// flood GetPage (view increments)
	go func() {
		for i := 0; i < 80; i++ {
			_, _, _ = svc.GetPage(uid, string(model.UserRoleUser), page.ID)
		}
		close(done)
	}()

	var lastBody string
	for i := 0; i < 20; i++ {
		lastBody = "written-" + itoa(i)
		_, err := svc.UpdatePage(uid, string(model.UserRoleUser), page.ID, nil, &lastBody)
		if err != nil {
			t.Fatalf("update %d: %v", i, err)
		}
	}
	<-done

	// Read without relying on GetPage view side effects for body check:
	// one final GetPage after writers finish.
	got, _, err := svc.GetPage(uid, string(model.UserRoleUser), page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != lastBody {
		t.Fatalf("body clobbered by view bumps: want %q got %q", lastBody, got.Body)
	}
	hist, err := svc.ListHistory(uid, page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) == 0 || hist[0].Body != lastBody {
		t.Fatalf("latest history body want %q, hist0=%+v", lastBody, hist[0])
	}
	if got.Body != hist[0].Body {
		t.Fatalf("current body %q != latest history %q", got.Body, hist[0].Body)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [16]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

// Non-owners without write grant can still read body; can_write is false.
func TestLoggedInUserCanReadWithoutWritePermission(t *testing.T) {
	svc := newTestCompass()
	const owner int64 = 501
	const reader int64 = 502
	app, _ := svc.ApplyAuthor(owner, "o")
	_, _ = svc.ReviewAuthorApplication(1, string(model.UserRoleAdmin), app.ID, true, "")
	page, err := svc.CreateEssay(owner, CreateEssayInput{Title: "public essay", Body: "readable body"})
	if err != nil {
		t.Fatal(err)
	}

	got, canWrite, err := svc.GetPage(reader, string(model.UserRoleUser), page.ID)
	if err != nil {
		t.Fatalf("reader should read without author/write: %v", err)
	}
	if canWrite {
		t.Fatal("reader must not have write")
	}
	if got.Body != "readable body" {
		t.Fatalf("body want readable body, got %q", got.Body)
	}

	// history is also readable
	hist, err := svc.ListHistory(reader, page.ID)
	if err != nil || len(hist) == 0 {
		t.Fatalf("reader should list history: %v len=%d", err, len(hist))
	}
}

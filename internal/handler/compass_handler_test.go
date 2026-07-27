package handler

import (
	"bytes"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/docengine"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/service"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupCompassRouter(h *CompassHandler) *gin.Engine {
	r := gin.New()
	// simulate JWTAuth by injecting user from headers in tests
	inject := func(c *gin.Context) {
		if uid := c.GetHeader("X-Test-User"); uid != "" {
			id, _ := strconv.ParseInt(uid, 10, 64)
			c.Set(constant.GinUserID, id)
		}
		if role := c.GetHeader("X-Test-Role"); role != "" {
			c.Set(constant.GinUserRole, role)
		}
		c.Next()
	}
	g := r.Group("/compass")
	g.Use(inject)
	{
		g.GET("/feed", h.GetFeed)
		g.GET("/pages/:id", h.GetPage)
		g.PATCH("/pages/:id", h.UpdatePage)
		g.GET("/pages/:id/history", h.ListHistory)
		g.POST("/essays", h.CreateEssay)
		g.POST("/collections", h.CreateCollection)
		g.POST("/author/apply", h.ApplyAuthor)
		g.GET("/author/me", h.AuthorMe)
		g.POST("/author/applications/:id/review", h.ReviewAuthorApplication)
		g.POST("/pages/:id/edit-requests", h.RequestEdit)
		g.POST("/edit-requests/:id/review", h.ReviewEditRequest)
		g.GET("/courses/:courseId/root", h.GetCourseRoot)
	}
	return r
}

func doJSON(r *gin.Engine, method, path string, userID int64, role string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		req.Header.Set("X-Test-User", strconv.FormatInt(userID, 10))
	}
	if role != "" {
		req.Header.Set("X-Test-Role", role)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	return payload
}

func TestCompassHandlerUnauthenticatedFeed(t *testing.T) {
	h := NewCompassHandler(service.NewCompassService(docengine.NewMemoryStore()))
	r := setupCompassRouter(h)
	w := doJSON(r, http.MethodGet, "/compass/feed?tab=recent", 0, "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestCompassHandlerAuthorFlowEssayFeedHistoryEdit(t *testing.T) {
	h := NewCompassHandler(service.NewCompassService(docengine.NewMemoryStore()))
	r := setupCompassRouter(h)

	const author int64 = 7
	const admin int64 = 1

	// apply author
	w := doJSON(r, http.MethodPost, "/compass/author/apply", author, "user", map[string]any{"reason": "share notes"})
	if w.Code != http.StatusOK {
		t.Fatalf("apply: %d %s", w.Code, w.Body.String())
	}
	appPayload := decodeData(t, w)
	appData := appPayload["data"].(map[string]any)
	appIDStr, _ := appData["id"].(string)
	if appIDStr == "" {
		// may be number depending on encoding
		if n, ok := appData["id"].(float64); ok {
			appIDStr = strconv.FormatInt(int64(n), 10)
		}
	}

	// approve
	w = doJSON(r, http.MethodPost, "/compass/author/applications/"+appIDStr+"/review", admin, "admin",
		map[string]any{"approve": true, "remark": "ok"})
	if w.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", w.Code, w.Body.String())
	}

	// non-author blocked (fresh user)
	w = doJSON(r, http.MethodPost, "/compass/essays", 99, "user", map[string]any{"title": "x", "body": "y"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-author create want 403 got %d", w.Code)
	}

	// create essay
	w = doJSON(r, http.MethodPost, "/compass/essays", author, "user", map[string]any{
		"title": "handler essay", "body": "hello body",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create essay: %d %s", w.Code, w.Body.String())
	}
	essay := decodeData(t, w)["data"].(map[string]any)
	pageID := idString(essay["id"])

	// create collection
	w = doJSON(r, http.MethodPost, "/compass/collections", author, "user", map[string]any{
		"title": "col1", "description": "d",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create col: %d %s", w.Code, w.Body.String())
	}

	// feed recent
	w = doJSON(r, http.MethodGet, "/compass/feed?tab=recent&type=all", author, "user", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("feed: %d %s", w.Code, w.Body.String())
	}
	feed := decodeData(t, w)["data"].(map[string]any)
	items := feed["items"].([]any)
	if len(items) < 1 {
		t.Fatalf("feed empty")
	}

	// hot
	w = doJSON(r, http.MethodGet, "/compass/feed?tab=hot", author, "user", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("hot feed: %d", w.Code)
	}

	// history after write
	w = doJSON(r, http.MethodPatch, "/compass/pages/"+pageID, author, "user", map[string]any{
		"body": "updated body",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}
	w = doJSON(r, http.MethodGet, "/compass/pages/"+pageID+"/history", author, "user", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("history: %d %s", w.Code, w.Body.String())
	}
	hist := decodeData(t, w)["data"].(map[string]any)
	histItems := hist["items"].([]any)
	if len(histItems) < 2 {
		t.Fatalf("want history >=2, got %d", len(histItems))
	}

	// edit request flow
	const other int64 = 8
	w = doJSON(r, http.MethodPost, "/compass/pages/"+pageID+"/edit-requests", other, "user", map[string]any{
		"reason": "help edit",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("edit req: %d %s", w.Code, w.Body.String())
	}
	reqID := idString(decodeData(t, w)["data"].(map[string]any)["id"])

	// reject then still cannot write
	w = doJSON(r, http.MethodPost, "/compass/edit-requests/"+reqID+"/review", author, "user", map[string]any{
		"approve": false,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("reject edit: %d %s", w.Code, w.Body.String())
	}
	w = doJSON(r, http.MethodPatch, "/compass/pages/"+pageID, other, "user", map[string]any{"body": "nope"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403 after reject, got %d", w.Code)
	}

	// re-request and approve
	w = doJSON(r, http.MethodPost, "/compass/pages/"+pageID+"/edit-requests", other, "user", map[string]any{
		"reason": "again",
	})
	reqID = idString(decodeData(t, w)["data"].(map[string]any)["id"])
	w = doJSON(r, http.MethodPost, "/compass/edit-requests/"+reqID+"/review", author, "user", map[string]any{
		"approve": true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("approve edit: %d %s", w.Code, w.Body.String())
	}
	w = doJSON(r, http.MethodPatch, "/compass/pages/"+pageID, other, "user", map[string]any{"body": "collab ok"})
	if w.Code != http.StatusOK {
		t.Fatalf("collab write: %d %s", w.Code, w.Body.String())
	}
}

func TestCompassHandlerCourseRoot(t *testing.T) {
	h := NewCompassHandler(service.NewCompassService(docengine.NewMemoryStore()))
	r := setupCompassRouter(h)

	w := doJSON(r, http.MethodGet, "/compass/courses/12345/root?title=高数", 0, "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth course root want 401 got %d", w.Code)
	}

	w = doJSON(r, http.MethodGet, "/compass/courses/12345/root?title=高数", 3, "user", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("course root: %d %s", w.Code, w.Body.String())
	}
	data := decodeData(t, w)["data"].(map[string]any)
	pageID := idString(data["page_id"])
	if pageID == "" || pageID == "0" {
		t.Fatalf("missing page_id: %+v", data)
	}

	// stable
	w2 := doJSON(r, http.MethodGet, "/compass/courses/12345/root", 3, "user", nil)
	data2 := decodeData(t, w2)["data"].(map[string]any)
	if idString(data2["page_id"]) != pageID {
		t.Fatalf("root not stable")
	}
	_ = model.CompassSpaceCourses
}

func idString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func TestParseOptionalJSONInt64SnowflakeString(t *testing.T) {
	// > 2^53-1 — would lose precision if coerced via float64/Number in JS.
	const snowflake = "2081794825308340224"
	id, err := parseOptionalJSONInt64(json.RawMessage(`"` + snowflake + `"`))
	if err != nil {
		t.Fatal(err)
	}
	if id == nil || *id != 2081794825308340224 {
		t.Fatalf("want exact snowflake, got %v", id)
	}
	// JSON number form still works for smaller IDs
	id2, err := parseOptionalJSONInt64(json.RawMessage(`42`))
	if err != nil || id2 == nil || *id2 != 42 {
		t.Fatalf("number form: %v %v", id2, err)
	}
	id3, err := parseOptionalJSONInt64(json.RawMessage(`null`))
	if err != nil || id3 != nil {
		t.Fatalf("null form: %v %v", id3, err)
	}
}

func TestCompassHandlerCreateEssayAcceptsSnowflakeStringIDs(t *testing.T) {
	store := docengine.NewMemoryStore()
	h := NewCompassHandler(service.NewCompassService(store))
	r := setupCompassRouter(h)

	const author int64 = 11
	const admin int64 = 1
	w := doJSON(r, http.MethodPost, "/compass/author/apply", author, "user", map[string]any{"reason": "x"})
	appID := idString(decodeData(t, w)["data"].(map[string]any)["id"])
	_ = doJSON(r, http.MethodPost, "/compass/author/applications/"+appID+"/review", admin, "admin",
		map[string]any{"approve": true})

	// Pre-seed a collection page with a large snowflake-like id by creating via API then
	// posting essay with collection_id as string (the app client path).
	w = doJSON(r, http.MethodPost, "/compass/collections", author, "user", map[string]any{
		"title": "snow col", "description": "d",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create col: %d %s", w.Code, w.Body.String())
	}
	col := decodeData(t, w)["data"].(map[string]any)["collection"].(map[string]any)
	colID := idString(col["id"])
	if colID == "" {
		t.Fatalf("missing collection id: %+v", col)
	}

	// Send collection_id as JSON string (not number) — mirrors createCompassEssay after fix.
	w = doJSON(r, http.MethodPost, "/compass/essays", author, "user", map[string]any{
		"title":         "under col",
		"body":          "child",
		"collection_id": colID, // string in JSON
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create essay under collection with string id: %d %s", w.Code, w.Body.String())
	}
	page := decodeData(t, w)["data"].(map[string]any)
	parent := idString(page["parent_id"])
	rootID := idString(col["root_page_id"])
	if parent == "" || parent != rootID {
		t.Fatalf("parent_id want root %s got %s (page=%+v)", rootID, parent, page)
	}
}

func TestParseOptionalJSONInt64RejectsLossyFloatString(t *testing.T) {
	// Ensure we don't silently accept non-integer tokens
	_, err := parseOptionalJSONInt64(json.RawMessage(`"12.5"`))
	if err == nil {
		t.Fatal("expected error for non-integer")
	}
}

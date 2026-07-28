package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type CompassHandler struct {
	svc *service.CompassService
}

func NewCompassHandler(svc *service.CompassService) *CompassHandler {
	return &CompassHandler{svc: svc}
}

func (h *CompassHandler) userID(c *gin.Context) int64 {
	if v, ok := c.Get(constant.GinUserID); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

func (h *CompassHandler) userRole(c *gin.Context) string {
	if v, ok := c.Get(constant.GinUserRole); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (h *CompassHandler) mapErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCompassUnauthorized):
		respFail(c, http.StatusUnauthorized, "请先登录")
	case errors.Is(err, service.ErrCompassForbidden):
		respFail(c, http.StatusForbidden, "无权限")
	case errors.Is(err, service.ErrCompassNotFound):
		respFail(c, http.StatusNotFound, "资源不存在")
	case errors.Is(err, service.ErrCompassNotAuthor):
		respFail(c, http.StatusForbidden, "需要作者身份")
	case errors.Is(err, service.ErrCompassInvalidPayload):
		respFail(c, http.StatusBadRequest, "参数无效")
	case errors.Is(err, service.ErrCompassConflict):
		respFail(c, http.StatusConflict, "操作冲突")
	case errors.Is(err, service.ErrCompassNoWrite):
		respFail(c, http.StatusForbidden, "无编辑权限")
	default:
		failInternalWithLog(c, err)
	}
}

func respFail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": 1, "msg": msg, "data": nil})
}

func respOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": data})
}

// GET /compass/feed?tab=recent|hot&type=all|essay|collection|...
func (h *CompassHandler) GetFeed(c *gin.Context) {
	items, err := h.svc.GetFeed(h.userID(c), c.Query("tab"), c.Query("type"), queryInt(c, "limit", 20))
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"items": items})
}

// GET /compass/pages/:id
// Any logged-in user may read; can_write only reflects edit rights (owner/writer/reviewer).
func (h *CompassHandler) GetPage(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	page, canWrite, err := h.svc.GetPage(h.userID(c), h.userRole(c), id)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"page": page, "can_write": canWrite})
}

// PATCH /compass/pages/:id
func (h *CompassHandler) UpdatePage(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var body struct {
		Title *string `json:"title"`
		Body  *string `json:"body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	page, err := h.svc.UpdatePage(h.userID(c), h.userRole(c), id, body.Title, body.Body)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, page)
}

// GET /compass/pages/:id/history
func (h *CompassHandler) ListHistory(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	items, err := h.svc.ListHistory(h.userID(c), id)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"items": items})
}

// GET /compass/pages/:id/comments
func (h *CompassHandler) ListComments(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	items, err := h.svc.ListComments(h.userID(c), id)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"items": items})
}

// POST /compass/pages/:id/comments
func (h *CompassHandler) AddComment(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	item, err := h.svc.AddComment(h.userID(c), id, body.Body)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, item)
}

// GET /compass/tree?space=plaza&root_page_id=
func (h *CompassHandler) GetTree(c *gin.Context) {
	rootID, _ := strconv.ParseInt(c.Query("root_page_id"), 10, 64)
	tree, err := h.svc.GetTree(h.userID(c), c.Query("space"), rootID)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"tree": tree})
}

// POST /compass/essays
func (h *CompassHandler) CreateEssay(c *gin.Context) {
	// IDs accept decimal strings (preferred for snowflakes) or JSON numbers.
	var body struct {
		Title        string          `json:"title"`
		Body         string          `json:"body"`
		CollectionID json.RawMessage `json:"collection_id"`
		CourseID     json.RawMessage `json:"course_id"`
		ParentID     json.RawMessage `json:"parent_id"`
		SpaceKey     string          `json:"space_key"`
		ContentType  string          `json:"content_type"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	collectionID, err := parseOptionalJSONInt64(body.CollectionID)
	if err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	courseID, err := parseOptionalJSONInt64(body.CourseID)
	if err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	parentID, err := parseOptionalJSONInt64(body.ParentID)
	if err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	page, err := h.svc.CreateEssay(h.userID(c), service.CreateEssayInput{
		Title: body.Title, Body: body.Body,
		CollectionID: collectionID, CourseID: courseID,
		ParentID: parentID, SpaceKey: body.SpaceKey, ContentType: body.ContentType,
	})
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, page)
}

// POST /compass/collections
func (h *CompassHandler) CreateCollection(c *gin.Context) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		CoverURL    string `json:"cover_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	col, root, err := h.svc.CreateCollection(h.userID(c), service.CreateCollectionInput{
		Title: body.Title, Description: body.Description, CoverURL: body.CoverURL,
	})
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"collection": col, "root_page": root})
}

// GET /compass/collections/:id
func (h *CompassHandler) GetCollection(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	col, root, err := h.svc.GetCollection(h.userID(c), id)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"collection": col, "root_page": root})
}

// POST /compass/author/apply
func (h *CompassHandler) ApplyAuthor(c *gin.Context) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	app, err := h.svc.ApplyAuthor(h.userID(c), body.Reason)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, app)
}

// GET /compass/author/me
func (h *CompassHandler) AuthorMe(c *gin.Context) {
	isAuthor, latest, err := h.svc.AuthorStatus(h.userID(c))
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"is_author": isAuthor, "latest_application": latest})
}

// GET /compass/author/applications
func (h *CompassHandler) ListAuthorApplications(c *gin.Context) {
	items, err := h.svc.ListAuthorApplications(h.userRole(c), c.Query("status"))
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"items": items})
}

// POST /compass/author/applications/:id/review
func (h *CompassHandler) ReviewAuthorApplication(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var body struct {
		Approve bool   `json:"approve"`
		Remark  string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	app, err := h.svc.ReviewAuthorApplication(h.userID(c), h.userRole(c), id, body.Approve, body.Remark)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, app)
}

// POST /compass/pages/:id/edit-requests
func (h *CompassHandler) RequestEdit(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	req, err := h.svc.RequestEdit(h.userID(c), id, body.Reason)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, req)
}

// POST /compass/edit-requests/:id/review
func (h *CompassHandler) ReviewEditRequest(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var body struct {
		Approve bool   `json:"approve"`
		Remark  string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return
	}
	req, err := h.svc.ReviewEditRequest(h.userID(c), h.userRole(c), id, body.Approve, body.Remark)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, req)
}

// GET /compass/courses/:courseId/root  (lazy ensure)
func (h *CompassHandler) GetCourseRoot(c *gin.Context) {
	courseID, ok := parseIDParam(c, "courseId")
	if !ok {
		return
	}
	title := c.Query("title")
	page, err := h.svc.EnsureCourseRoot(h.userID(c), courseID, title)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	respOK(c, gin.H{"page": page, "page_id": page.ID})
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		respFail(c, http.StatusBadRequest, constant.BadRequestErr.Error())
		return 0, false
	}
	return id, true
}

func queryInt(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// parseOptionalJSONInt64 accepts null/empty, JSON number, or decimal string
// so snowflake IDs beyond JS safe-integer range stay exact when sent as strings.
func parseOptionalJSONInt64(raw json.RawMessage) (*int64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	s := strings.TrimSpace(string(raw))
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

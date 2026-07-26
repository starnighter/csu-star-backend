package handler

import (
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"csu-star-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type WikiHandler struct {
	wikiSvc *service.WikiService
}

func NewWikiHandler(wikiSvc *service.WikiService) *WikiHandler {
	return &WikiHandler{wikiSvc: wikiSvc}
}

// ---------- 公开接口 ----------

func (h *WikiHandler) GetTree(c *gin.Context) {
	tree, err := h.wikiSvc.GetTree()
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, tree)
}

func (h *WikiHandler) GetDoc(c *gin.Context) {
	detail, err := h.wikiSvc.GetDoc(c.Param("section"), c.Param("slug"))
	switch {
	case errors.Is(err, service.ErrWikiNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "文档不存在")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, detail)
	}
}

// ---------- admin:分类 ----------

func (h *WikiHandler) ListCategories(c *gin.Context) {
	var r req.WikiCategoryListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, err := h.wikiSvc.ListCategories(r.Section)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items})
}

func (h *WikiHandler) CreateCategory(c *gin.Context) {
	var r req.WikiCategoryInput
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	item, err := h.wikiSvc.CreateCategory(currentUserID(c), r.Section, r.Name, r.SortOrder, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrWikiInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "分类参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *WikiHandler) UpdateCategory(c *gin.Context) {
	id, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.WikiCategoryInput
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	item, err := h.wikiSvc.UpdateCategory(currentUserID(c), id, r.Name, r.SortOrder, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrWikiNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "分类不存在")
	case errors.Is(err, service.ErrWikiInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "分类参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *WikiHandler) DeleteCategory(c *gin.Context) {
	id, ok := parsePositiveID(c)
	if !ok {
		return
	}
	err := h.wikiSvc.DeleteCategory(currentUserID(c), id, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrWikiNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "分类不存在")
	case errors.Is(err, service.ErrWikiCategoryNotEmpty):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "分类下仍有文档,无法删除")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "删除成功")
	}
}

func (h *WikiHandler) ReorderCategories(c *gin.Context) {
	ids, ok := bindReorderIDs(c)
	if !ok {
		return
	}
	if err := h.wikiSvc.ReorderCategories(currentUserID(c), ids, parseIP(c.ClientIP())); err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.SuccessMsg(c, "排序已更新")
}

// ---------- admin:文档 ----------

func (h *WikiHandler) ListDocs(c *gin.Context) {
	var r req.WikiDocListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	query := repo.WikiDocListQuery{
		Section: r.Section,
		Keyword: r.Keyword,
		Status:  r.Status,
		Page:    r.Page,
		Size:    r.Size,
	}
	if r.CategoryID != "" {
		// category_id=0 表示筛选板块直属文档
		id, err := strconv.ParseInt(r.CategoryID, 10, 64)
		if err != nil || id < 0 {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
		query.CategoryID = &id
	}
	items, total, err := h.wikiSvc.ListDocs(query)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *WikiHandler) GetDocByID(c *gin.Context) {
	id, ok := parsePositiveID(c)
	if !ok {
		return
	}
	doc, err := h.wikiSvc.GetDocByID(id)
	switch {
	case errors.Is(err, service.ErrWikiNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "文档不存在")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, doc)
	}
}

func (h *WikiHandler) CreateDoc(c *gin.Context) {
	var r req.WikiDocCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	categoryID, ok := parseOptionalID(c, r.CategoryID)
	if !ok {
		return
	}
	item, err := h.wikiSvc.CreateDoc(currentUserID(c), r.Section, categoryID, r.Slug, r.Title, r.Content, r.SortOrder, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrWikiInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "文档参数无效")
	case errors.Is(err, service.ErrWikiSlugConflict):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "slug 已存在,请更换")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *WikiHandler) UpdateDoc(c *gin.Context) {
	id, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.WikiDocUpdateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	update := service.WikiDocUpdate{
		Slug:      r.Slug,
		Title:     r.Title,
		Content:   r.Content,
		SortOrder: r.SortOrder,
	}
	if r.CategoryID != nil {
		if *r.CategoryID == "" {
			update.ClearCategory = true
		} else {
			categoryID, ok := parseOptionalID(c, *r.CategoryID)
			if !ok {
				return
			}
			update.CategoryID = categoryID
		}
	}
	item, err := h.wikiSvc.UpdateDoc(currentUserID(c), id, update, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrWikiNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "文档不存在")
	case errors.Is(err, service.ErrWikiInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "文档参数无效")
	case errors.Is(err, service.ErrWikiSlugConflict):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "slug 已存在,请更换")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *WikiHandler) PublishDoc(c *gin.Context) {
	h.setDocPublished(c, true)
}

func (h *WikiHandler) UnpublishDoc(c *gin.Context) {
	h.setDocPublished(c, false)
}

func (h *WikiHandler) setDocPublished(c *gin.Context, published bool) {
	id, ok := parsePositiveID(c)
	if !ok {
		return
	}
	item, err := h.wikiSvc.SetDocPublished(currentUserID(c), id, published, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrWikiNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "文档不存在")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *WikiHandler) ReorderDocs(c *gin.Context) {
	ids, ok := bindReorderIDs(c)
	if !ok {
		return
	}
	if err := h.wikiSvc.ReorderDocs(currentUserID(c), ids, parseIP(c.ClientIP())); err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.SuccessMsg(c, "排序已更新")
}

func (h *WikiHandler) DeleteDoc(c *gin.Context) {
	id, ok := parsePositiveID(c)
	if !ok {
		return
	}
	err := h.wikiSvc.DeleteDoc(currentUserID(c), id, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrWikiNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "文档不存在")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "删除成功")
	}
}

// ---------- admin:图片上传 ----------

const wikiImageMaxSize = 5 << 20 // 5MB

// 白名单 + 限体积,防止该接口被当成通用文件床。
var wikiImageExtWhitelist = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

func (h *WikiHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	if file.Size > wikiImageMaxSize {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "图片不能超过 5MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if _, ok := wikiImageExtWhitelist[ext]; !ok {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "仅支持 png/jpg/gif/webp 图片")
		return
	}

	info, err := utils.TencentCosUpload(file, constant.TencentCosWikiImagesKeyPrefix)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"url": info.FileUrl, "file_key": info.FileKey})
}

// ---------- 内部工具 ----------

func bindReorderIDs(c *gin.Context) ([]int64, bool) {
	var r req.WikiReorderReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return nil, false
	}
	ids := make([]int64, 0, len(r.IDs))
	for _, raw := range r.IDs {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

func parseOptionalID(c *gin.Context, raw string) (*int64, bool) {
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return nil, false
	}
	return &id, true
}

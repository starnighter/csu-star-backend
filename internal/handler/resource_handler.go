package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	resourceSvc *service.ResourceService
}

func NewResourceHandler(svc *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{resourceSvc: svc}
}

func (h *ResourceHandler) GetResources(c *gin.Context) {
	var r req.ResourceListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, total, err := h.resourceSvc.ListResources(repo.ResourceListQuery{
		Q:            r.Q,
		CourseID:     r.CourseID,
		ResourceType: r.ResourceType,
		Sort:         r.Sort,
		Page:         r.Page,
		Size:         r.Size,
	})
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *ResourceHandler) GetResourceDetail(c *gin.Context) {
	resourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || resourceID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var userID int64
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}

	detail, err := h.resourceSvc.GetResourceDetail(resourceID, userID)
	switch {
	case errors.Is(err, service.ErrResourceNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "资源不存在")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, detail)
}

func (h *ResourceHandler) CreateResource(c *gin.Context) {
	title := c.PostForm("title")
	description := c.PostForm("description")
	courseIDStr := c.PostForm("course_id")
	resourceType := c.PostForm("resource_type")
	semester := c.PostForm("semester")

	courseID, err := strconv.ParseInt(courseIDStr, 10, 64)
	if title == "" || courseID <= 0 || resourceType == "" || err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	form, err := c.MultipartForm()
	if err != nil || form == nil || len(form.File["files"]) == 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "请至少上传一个文件")
		return
	}

	uploadedFiles, err := service.UploadResourceFiles(form.File["files"])
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	resource, files, err := h.resourceSvc.CreateResource(userID, title, description, courseID, resourceType, semester, uploadedFiles)
	switch {
	case errors.Is(err, service.ErrResourceCourseNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case errors.Is(err, service.ErrInvalidResourceFile):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "资源文件无效")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{
		"resource_id": resource.ID,
		"status":      resource.Status,
		"files":       files,
	})
}

func (h *ResourceHandler) SubmitResource(c *gin.Context) {
	resourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || resourceID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	err = h.resourceSvc.SubmitResource(userID, resourceID)
	switch {
	case errors.Is(err, service.ErrResourceNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "资源不存在或不可提交")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{"status": "pending"})
}

func (h *ResourceHandler) DownloadResource(c *gin.Context) {
	resourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || resourceID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.ResourceDownloadReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	result, err := h.resourceSvc.DownloadResource(userID, resourceID, r.FileID, service.ParseClientIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrResourceNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "资源不存在")
		return
	case errors.Is(err, service.ErrInsufficientPoints):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "积分不足")
		return
	case errors.Is(err, service.ErrNoFreeDownloadChance):
		resp.FailWithCode(c, http.StatusBadRequest, 10005, "免费下载次数已用完，请先绑定邮箱")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, result)
}

func (h *ResourceHandler) GetMyUploads(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	userID := c.MustGet(constant.GinUserID).(int64)
	items, total, err := h.resourceSvc.ListMyUploads(userID, page, size)
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

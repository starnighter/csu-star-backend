package handler

import (
	"csu-star-backend/internal/constant"
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

func (h *ResourceHandler) GetResourceDetail(c *gin.Context) {
	resourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || resourceID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var userID int64
	var userRole string
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}
	if v, ok := c.Get(constant.GinUserRole); ok {
		userRole = v.(string)
	}

	detail, err := h.resourceSvc.GetResourceDetail(resourceID, userID, userRole)
	switch {
	case errors.Is(err, service.ErrResourceNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "资源不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, detail)
}

func (h *ResourceHandler) CreateResource(c *gin.Context) {
	var r req.ResourceCreateInputReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	courseID, err := parseStringID(r.CourseID)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	uploadedFiles := make([]service.UploadedResourceFile, 0, len(r.Files))
	for _, file := range r.Files {
		uploadedFiles = append(uploadedFiles, service.NewUploadedResourceFile(file.Filename, file.SizeBytes, file.Mime))
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	data, err := h.resourceSvc.PrepareResourceUpload(
		userID,
		r.Title,
		r.Description,
		courseID,
		r.ResourceType,
		uploadedFiles,
	)
	switch {
	case errors.Is(err, service.ErrResourceEmailNotVerified):
		resp.FailWithCode(c, http.StatusForbidden, constant.EmailNotVerifiedUploadErr.Code, constant.EmailNotVerifiedUploadErr.Msg)
		return
	case errors.Is(err, service.ErrResourceRateLimited):
		resp.FailWithData(c, http.StatusTooManyRequests, constant.TooManyRequestsErr.Code, constant.TooManyRequestsErr.Msg, gin.H{"scope": "resource_prepare"})
		return
	case errors.Is(err, service.ErrResourceUserBanned):
		resp.FailWithCode(c, http.StatusForbidden, constant.UserAutoBannedErr.Code, constant.UserAutoBannedErr.Msg)
		return
	case errors.Is(err, service.ErrResourceCourseNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case errors.Is(err, service.ErrResourceUploadTooLarge):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "单个资源上传总大小不能超过 300MB")
		return
	case errors.Is(err, service.ErrInvalidResourceFile):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "资源文件无效")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, data)
}

func (h *ResourceHandler) FinalizeResourceUpload(c *gin.Context) {
	var r req.ResourceFinalizeUploadReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	data, err := h.resourceSvc.FinalizeResourceUpload(userID, r.UploadSessionID)
	switch {
	case errors.Is(err, service.ErrResourceRateLimited):
		resp.FailWithData(c, http.StatusTooManyRequests, constant.TooManyRequestsErr.Code, constant.TooManyRequestsErr.Msg, gin.H{"scope": "resource_finalize"})
		return
	case errors.Is(err, service.ErrResourceUploadSessionNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "上传会话不存在或已过期")
		return
	case errors.Is(err, service.ErrResourceUploadSessionForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权操作该上传会话")
		return
	case errors.Is(err, service.ErrResourceUploadIncomplete):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "资源文件尚未全部上传成功")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, data)
}

func (h *ResourceHandler) AbortResourceUpload(c *gin.Context) {
	var r req.ResourceAbortUploadReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	err := h.resourceSvc.AbortResourceUpload(userID, r.UploadSessionID)
	switch {
	case errors.Is(err, service.ErrResourceRateLimited):
		resp.FailWithData(c, http.StatusTooManyRequests, constant.TooManyRequestsErr.Code, constant.TooManyRequestsErr.Msg, gin.H{"scope": "resource_abort"})
		return
	case errors.Is(err, service.ErrResourceUploadSessionNotFound):
		resp.SuccessMsg(c, "上传会话不存在或已过期")
		return
	case errors.Is(err, service.ErrResourceUploadSessionForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权操作该上传会话")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.SuccessMsg(c, "上传会话已清理")
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
	case errors.Is(err, service.ErrResourceRateLimited):
		resp.FailWithData(c, http.StatusTooManyRequests, constant.TooManyRequestsErr.Code, constant.TooManyRequestsErr.Msg, gin.H{"scope": "resource_download"})
		return
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
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, result)
}

func (h *ResourceHandler) GetMyUploads(c *gin.Context) {
	var r req.MyResourcesReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	items, total, err := h.resourceSvc.ListMyResources(userID, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *ResourceHandler) DeleteResource(c *gin.Context) {
	resourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || resourceID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	err = h.resourceSvc.DeleteResource(userID, userRole, resourceID)
	switch {
	case errors.Is(err, service.ErrResourceNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "资源不存在")
		return
	case errors.Is(err, service.ErrResourceAlreadyDeleted):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "资源已删除")
		return
	case errors.Is(err, service.ErrResourceForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权限删除该资源")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.SuccessMsg(c, "删除成功")
}

func (h *ResourceHandler) UpdateResource(c *gin.Context) {
	resourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || resourceID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	var r req.ResourceUpdateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	courseID, err := parseStringID(r.CourseID)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	item, err := h.resourceSvc.UpdateResource(userID, userRole, resourceID, r.Title, r.Description, courseID, r.ResourceType)
	switch {
	case errors.Is(err, service.ErrResourceNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "资源不存在")
		return
	case errors.Is(err, service.ErrResourceAlreadyDeleted):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "资源已删除")
		return
	case errors.Is(err, service.ErrResourceForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权限修改该资源")
		return
	case errors.Is(err, service.ErrResourceCourseNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

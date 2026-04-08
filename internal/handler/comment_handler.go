package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	commentSvc *service.CommentService
}

func NewCommentHandler(svc *service.CommentService) *CommentHandler {
	return &CommentHandler{commentSvc: svc}
}

func (h *CommentHandler) GetResourceComments(c *gin.Context) {
	h.getCommentsByTarget(c, model.CommentTargetTypeResource, "id")
}

func (h *CommentHandler) GetTeacherComments(c *gin.Context) {
	h.getCommentsByTarget(c, model.CommentTargetTypeTeacher, "id")
}

func (h *CommentHandler) GetCourseComments(c *gin.Context) {
	h.getCommentsByTarget(c, model.CommentTargetTypeCourse, "id")
}

func (h *CommentHandler) CreateResourceComment(c *gin.Context) {
	h.createCommentByTarget(c, model.CommentTargetTypeResource, "id")
}

func (h *CommentHandler) CreateTeacherComment(c *gin.Context) {
	h.createCommentByTarget(c, model.CommentTargetTypeTeacher, "id")
}

func (h *CommentHandler) CreateCourseComment(c *gin.Context) {
	h.createCommentByTarget(c, model.CommentTargetTypeCourse, "id")
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || commentID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)

	err = h.commentSvc.DeleteComment(userID, userRole, commentID)
	switch {
	case errors.Is(err, service.ErrCommentNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "评论不存在")
		return
	case errors.Is(err, service.ErrCommentForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权删除该评论")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.SuccessMsg(c, "删除评论成功")
}

func (h *CommentHandler) UpdateComment(c *gin.Context) {
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || commentID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	var r req.CommentUpdateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	item, err := h.commentSvc.UpdateComment(userID, commentID, r.Content)
	switch {
	case errors.Is(err, service.ErrCommentNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "评论不存在")
		return
	case errors.Is(err, service.ErrCommentForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权修改该评论")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *CommentHandler) getCommentsByTarget(c *gin.Context, targetType model.CommentTargetType, param string) {
	targetID, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || targetID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	var r req.CommentListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var userID int64
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}

	items, total, err := h.commentSvc.ListComments(targetType, targetID, r.Sort, r.Page, r.Size, userID)
	switch {
	case errors.Is(err, service.ErrCommentTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "目标不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *CommentHandler) createCommentByTarget(c *gin.Context, targetType model.CommentTargetType, param string) {
	targetID, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || targetID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	var r req.CommentCreateReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	var parentID int64
	if r.ParentID != "" {
		parentID, err = parseStringID(r.ParentID)
		if err != nil {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
	}
	var replyToCommentID int64
	if r.ReplyToCommentID != "" {
		replyToCommentID, err = parseStringID(r.ReplyToCommentID)
		if err != nil {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
	}
	userID := c.MustGet(constant.GinUserID).(int64)

	item, err := h.commentSvc.CreateComment(userID, targetType, targetID, r.Content, parentID, replyToCommentID)
	switch {
	case errors.Is(err, service.ErrCommentTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "目标不存在")
		return
	case errors.Is(err, service.ErrCommentReplyInvalid):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "回复关系无效")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, item)
}

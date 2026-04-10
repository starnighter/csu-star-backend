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

type MiscHandler struct {
	miscSvc *service.MiscService
}

func NewMiscHandler(svc *service.MiscService) *MiscHandler {
	return &MiscHandler{miscSvc: svc}
}

func (h *MiscHandler) GetMe(c *gin.Context) {
	userID := c.MustGet(constant.GinUserID).(int64)
	item, err := h.miscSvc.GetMe(userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *MiscHandler) UpdateMe(c *gin.Context) {
	var r req.UpdateMeReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	item, err := h.miscSvc.UpdateMe(userID, r.Nickname, r.AvatarURL, r.DepartmentID, r.Grade)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *MiscHandler) GetEmailStatus(c *gin.Context) {
	userID := c.MustGet(constant.GinUserID).(int64)
	item, err := h.miscSvc.GetMe(userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"email": item.Email, "email_verified": item.EmailVerified, "free_download_count": item.FreeDownloadCount})
}

func (h *MiscHandler) GetMyInviteCode(c *gin.Context) {
	userID := c.MustGet(constant.GinUserID).(int64)
	item, err := h.miscSvc.GetMyInviteCode(userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *MiscHandler) DailyCheckin(c *gin.Context) {
	userID := c.MustGet(constant.GinUserID).(int64)
	balance, err := h.miscSvc.DailyCheckin(userID)
	switch {
	case errors.Is(err, service.ErrAlreadyCheckedIn):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "今天已经签到过了")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"points": balance})
}

func (h *MiscHandler) GetMyDownloads(c *gin.Context) {
	var r req.PaginationReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	items, total, err := h.miscSvc.ListMyDownloads(userID, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *MiscHandler) GetMyFavorites(c *gin.Context) {
	var r req.FavoriteListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	items, total, err := h.miscSvc.ListMyFavorites(userID, r.TargetType, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *MiscHandler) GetMyPoints(c *gin.Context) {
	var r req.PaginationReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	items, total, err := h.miscSvc.ListMyPoints(userID, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *MiscHandler) GetMyContributions(c *gin.Context) {
	userID := c.MustGet(constant.GinUserID).(int64)
	item, err := h.miscSvc.GetMyContributionSummary(userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *MiscHandler) GetAnnouncements(c *gin.Context) {
	items, err := h.miscSvc.ListAnnouncements()
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items})
}

func (h *MiscHandler) GetShowcaseStats(c *gin.Context) {
	item, err := h.miscSvc.GetShowcaseStats()
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *MiscHandler) CreateFeedback(c *gin.Context) {
	var r req.FeedbackCreateReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	if err := h.miscSvc.CreateFeedback(userID, r.Type, r.Title, r.Content, r.Files); err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.SuccessMsg(c, "反馈提交成功")
}

func (h *MiscHandler) CreateSupplementRequest(c *gin.Context) {
	var r req.SupplementRequestCreateReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	item, err := h.miscSvc.CreateSupplementRequest(
		userID,
		r.RequestType,
		r.Contact,
		r.TeacherName,
		r.DepartmentID,
		r.RelatedCourseName,
		r.RelatedCourseIDs,
		r.RelatedCourseNames,
		r.RelatedTeacherIDs,
		r.RelatedTeacherNames,
		r.CourseName,
		r.CourseType,
		r.Remark,
	)
	if err != nil {
		if errors.Is(err, service.ErrSupplementRequestInvalidPayload) {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "补录申请参数不完整或格式错误")
			return
		}
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, item)
}

func (h *MiscHandler) CreateReport(c *gin.Context) {
	var r req.ReportCreateReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	targetID, err := parseStringID(r.TargetID)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	if err := h.miscSvc.CreateReport(userID, r.TargetType, targetID, r.Reason, r.Description); err != nil {
		if err == service.ErrSocialTargetNotFound {
			resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "目标不存在")
			return
		}
		failInternalWithLog(c, err)
		return
	}
	resp.SuccessMsg(c, "举报提交成功")
}

func (h *MiscHandler) CreateCorrection(c *gin.Context) {
	var r req.CorrectionCreateReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	targetID, err := parseStringID(r.TargetID)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	if err := h.miscSvc.CreateCorrection(userID, r.TargetType, targetID, r.Field, r.SuggestedValue); err != nil {
		if err == service.ErrSocialTargetNotFound {
			resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "目标不存在")
			return
		}
		failInternalWithLog(c, err)
		return
	}
	resp.SuccessMsg(c, "纠错提交成功")
}

func (h *MiscHandler) Search(c *gin.Context) {
	var r req.SearchReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	var userID int64
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}
	items, total, err := h.miscSvc.Search(userID, r.Q, r.Type, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *MiscHandler) GetNotifications(c *gin.Context) {
	var r req.NotificationListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	items, total, err := h.miscSvc.ListNotifications(userID, r.IsRead, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *MiscHandler) GetHomeNotificationSummary(c *gin.Context) {
	userID := c.MustGet(constant.GinUserID).(int64)
	summary, err := h.miscSvc.ListHomeNotificationSummary(userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, summary)
}

func (h *MiscHandler) GetUnreadCount(c *gin.Context) {
	userID := c.MustGet(constant.GinUserID).(int64)
	count, err := h.miscSvc.CountUnreadNotifications(userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"count": count})
}

func (h *MiscHandler) MarkNotificationRead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	if err := h.miscSvc.MarkNotificationRead(userID, id); err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.SuccessMsg(c, "标记成功")
}

func (h *MiscHandler) MarkAllNotificationsRead(c *gin.Context) {
	userID := c.MustGet(constant.GinUserID).(int64)
	if err := h.miscSvc.MarkAllNotificationsRead(userID); err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.SuccessMsg(c, "全部已读")
}

func (h *MiscHandler) ListSupplementRequests(c *gin.Context) {
	var r req.SupplementRequestListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, total, err := h.miscSvc.ListSupplementRequests(repo.SupplementRequestListQuery{
		Status:      r.Status,
		RequestType: r.RequestType,
		Keyword:     r.Keyword,
		Page:        r.Page,
		Size:        r.Size,
	})
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *MiscHandler) GetSupplementRequest(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	item, err := h.miscSvc.GetSupplementRequest(id)
	if err != nil {
		if errors.Is(err, service.ErrSupplementRequestNotFound) {
			resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "补录申请不存在")
			return
		}
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, item)
}

func (h *MiscHandler) ApproveSupplementRequest(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.SupplementRequestReviewReq
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindBodyWithJSON(&r); err != nil {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
	}

	operatorID := c.MustGet(constant.GinUserID).(int64)
	item, err := h.miscSvc.ApproveSupplementRequest(operatorID, id, r.ReviewNote)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplementRequestNotFound):
			resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "补录申请不存在")
			return
		case errors.Is(err, service.ErrSupplementRequestAlreadyReviewed):
			resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "补录申请已处理")
			return
		case errors.Is(err, service.ErrSupplementRequestInvalidPayload):
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "补录申请数据不合法")
			return
		default:
			failInternalWithLog(c, err)
			return
		}
	}

	resp.Success(c, item)
}

func (h *MiscHandler) RejectSupplementRequest(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.SupplementRequestReviewReq
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindBodyWithJSON(&r); err != nil {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
	}

	operatorID := c.MustGet(constant.GinUserID).(int64)
	item, err := h.miscSvc.RejectSupplementRequest(operatorID, id, r.ReviewNote)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplementRequestNotFound):
			resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "补录申请不存在")
			return
		case errors.Is(err, service.ErrSupplementRequestAlreadyReviewed):
			resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "补录申请已处理")
			return
		case errors.Is(err, service.ErrSupplementRequestReviewNoteMissing):
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "拒绝时必须填写审核备注")
			return
		default:
			failInternalWithLog(c, err)
			return
		}
	}

	resp.Success(c, item)
}

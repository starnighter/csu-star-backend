package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminSvc *service.AdminService
}

var errInvalidAdminTime = errors.New("invalid admin time")

func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{adminSvc: svc}
}

func (h *AdminHandler) GetStatistics(c *gin.Context) {
	item, err := h.adminSvc.GetStatistics()
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *AdminHandler) ListReports(c *gin.Context) {
	var r req.AdminReportListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListReports(r.Status, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) HandleReport(c *gin.Context) {
	reportID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminReportHandleReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	err := h.adminSvc.HandleReport(reportID, currentUserID(c), r.Action, r.Remark, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "举报不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "举报已处理")
	case errors.Is(err, service.ErrAdminInvalidAction):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "处理动作无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "处理成功")
	}
}

func (h *AdminHandler) ListCorrections(c *gin.Context) {
	var r req.AdminCorrectionListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListCorrections(r.Status, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) HandleCorrection(c *gin.Context) {
	correctionID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminCorrectionHandleReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	err := h.adminSvc.HandleCorrection(correctionID, currentUserID(c), r.Action, r.Remark, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "纠错记录不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "纠错记录已处理")
	case errors.Is(err, service.ErrAdminInvalidAction), errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "纠错处理参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "处理成功")
	}
}

func (h *AdminHandler) ListFeedbacks(c *gin.Context) {
	var r req.AdminFeedbackListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListFeedbacks(r.Status, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) ReplyFeedback(c *gin.Context) {
	feedbackID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminFeedbackReplyReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	err := h.adminSvc.ReplyFeedback(feedbackID, currentUserID(c), r.Reply, r.Status, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "反馈不存在")
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "回复内容无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "回复成功")
	}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var r req.AdminUserListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListUsers(r.Status, r.Role, r.Keyword, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) ListUserViolations(c *gin.Context) {
	userID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminPaginationReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListUserViolations(userID, int64(r.Page), int64(r.Size))
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var r req.AdminUserCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	item, err := h.adminSvc.CreateUser(currentUserID(c), r.Email, r.Password, r.Nickname, r.AvatarURL, r.Role, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "用户已存在")
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "用户注册参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *AdminHandler) BanUser(c *gin.Context) {
	userID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var payload struct {
		Reason string `json:"reason" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	err := h.adminSvc.BanUser(userID, currentUserID(c), payload.Reason, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "用户不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "用户已被封禁")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "封禁成功")
	}
}

func (h *AdminHandler) UnbanUser(c *gin.Context) {
	userID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	err := h.adminSvc.UnbanUser(userID, currentUserID(c), parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "用户不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "用户未被封禁")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "解封成功")
	}
}

func (h *AdminHandler) AdjustUserPoints(c *gin.Context) {
	userID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminUserAdjustPointsReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	balance, err := h.adminSvc.AdjustUserPoints(userID, currentUserID(c), r.Delta, r.Reason, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "用户不存在")
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "积分调整参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, gin.H{"current_points": balance})
	}
}

func (h *AdminHandler) SendUserNotification(c *gin.Context) {
	userID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminUserNotificationReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	err := h.adminSvc.SendUserNotification(
		userID,
		currentUserID(c),
		r.Title,
		r.Content,
		model.NotificationResult(strings.TrimSpace(r.Result)),
		parseIP(c.ClientIP()),
	)
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "用户不存在")
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "通知参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "发送成功")
	}
}

func (h *AdminHandler) ListAnnouncements(c *gin.Context) {
	var r req.AdminAnnouncementListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListAnnouncements(r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) CreateAnnouncement(c *gin.Context) {
	var r req.AdminAnnouncementInput
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	input, err := buildAnnouncementModel(r)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "公告参数无效")
		return
	}
	item, err := h.adminSvc.CreateAnnouncement(currentUserID(c), input, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "公告参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *AdminHandler) UpdateAnnouncement(c *gin.Context) {
	announcementID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminAnnouncementInput
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	updates, err := buildAnnouncementUpdates(r)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "公告参数无效")
		return
	}
	item, err := h.adminSvc.UpdateAnnouncement(currentUserID(c), announcementID, updates, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "公告不存在")
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "公告参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *AdminHandler) DeleteAnnouncement(c *gin.Context) {
	announcementID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	err := h.adminSvc.DeleteAnnouncement(currentUserID(c), announcementID, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "公告不存在")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "删除成功")
	}
}

func (h *AdminHandler) ListCourses(c *gin.Context) {
	var r req.AdminCourseListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListCourses(r.Status, r.CourseType, r.Keyword, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) CreateCourse(c *gin.Context) {
	var r req.AdminCourseInput
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	item, err := h.adminSvc.CreateCourse(currentUserID(c), &model.Courses{
		Name:        strings.TrimSpace(r.Name),
		CourseType:  model.CourseType(r.CourseType),
		Description: r.Description,
		Credits:     derefFloat64(r.Credits),
	}, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "课程参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *AdminHandler) UpdateCourse(c *gin.Context) {
	courseID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminCourseUpdateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	updates := map[string]interface{}{}
	if strings.TrimSpace(r.Name) != "" {
		updates["name"] = strings.TrimSpace(r.Name)
	}
	if r.CourseType != "" {
		updates["course_type"] = model.CourseType(r.CourseType)
	}
	if r.Description != nil {
		updates["description"] = *r.Description
	}
	if r.Credits != nil {
		updates["credits"] = *r.Credits
	}
	item, err := h.adminSvc.UpdateCourse(currentUserID(c), courseID, updates, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "课程参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *AdminHandler) DeleteCourse(c *gin.Context) {
	courseID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	err := h.adminSvc.DeleteCourse(currentUserID(c), courseID, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "课程已删除")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "删除成功")
	}
}

func (h *AdminHandler) ListCourseRelations(c *gin.Context) {
	courseID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	items, err := h.adminSvc.ListCourseRelations(courseID)
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, gin.H{"items": items})
	}
}

func (h *AdminHandler) AddCourseRelation(c *gin.Context) {
	courseID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminCourseRelationInputReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	teacherID, err := parseStringID(r.TeacherID)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	item, err := h.adminSvc.AddCourseRelation(currentUserID(c), courseID, teacherID, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程或教师不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "该教师与课程已关联")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *AdminHandler) RemoveCourseRelation(c *gin.Context) {
	courseID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	teacherID, err := strconv.ParseInt(c.Param("teacherId"), 10, 64)
	if err != nil || teacherID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	err = h.adminSvc.RemoveCourseRelation(currentUserID(c), courseID, teacherID, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程、教师或关联不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "该教师与课程当前未处于关联状态")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "取消关联成功")
	}
}

func (h *AdminHandler) ListTeachers(c *gin.Context) {
	var r req.AdminTeacherListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListTeachers(r.Status, r.Keyword, r.DepartmentID, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) CreateTeacher(c *gin.Context) {
	var r req.AdminTeacherInput
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	item, err := h.adminSvc.CreateTeacher(currentUserID(c), &model.Teachers{
		Name:         strings.TrimSpace(r.Name),
		DepartmentID: r.DepartmentID,
		Title:        strings.TrimSpace(r.Title),
		AvatarUrl:    strings.TrimSpace(r.AvatarURL),
	}, r.Bio, r.TutorType, r.HomepageURL, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "教师参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *AdminHandler) UpdateTeacher(c *gin.Context) {
	teacherID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminTeacherUpdateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	updates := map[string]interface{}{}
	var bio *string
	var tutorType *string
	var homepageURL *string
	if strings.TrimSpace(r.Name) != "" {
		updates["name"] = strings.TrimSpace(r.Name)
	}
	if r.DepartmentID != nil {
		updates["department_id"] = *r.DepartmentID
	}
	if r.Title != nil {
		updates["title"] = strings.TrimSpace(*r.Title)
	}
	if r.AvatarURL != nil {
		updates["avatar_url"] = strings.TrimSpace(*r.AvatarURL)
	}
	if r.Bio != nil {
		bio = r.Bio
	}
	if r.TutorType != nil {
		tutorType = r.TutorType
	}
	if r.HomepageURL != nil {
		homepageURL = r.HomepageURL
	}
	item, err := h.adminSvc.UpdateTeacher(currentUserID(c), teacherID, updates, bio, tutorType, homepageURL, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "教师参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *AdminHandler) DeleteTeacher(c *gin.Context) {
	teacherID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	err := h.adminSvc.DeleteTeacher(currentUserID(c), teacherID, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "教师已删除")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "删除成功")
	}
}

func (h *AdminHandler) ListTeacherRelations(c *gin.Context) {
	teacherID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	items, err := h.adminSvc.ListTeacherRelations(teacherID)
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, gin.H{"items": items})
	}
}

func (h *AdminHandler) AddTeacherRelation(c *gin.Context) {
	teacherID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.AdminTeacherRelationInputReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	courseID, err := parseStringID(r.CourseID)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	item, err := h.adminSvc.AddCourseRelation(currentUserID(c), courseID, teacherID, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师或课程不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "该课程与教师已关联")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *AdminHandler) RemoveTeacherRelation(c *gin.Context) {
	teacherID, ok := parsePositiveID(c)
	if !ok {
		return
	}
	courseID, err := strconv.ParseInt(c.Param("courseId"), 10, 64)
	if err != nil || courseID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	err = h.adminSvc.RemoveCourseRelation(currentUserID(c), courseID, teacherID, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师、课程或关联不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "该课程与教师当前未处于关联状态")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "取消关联成功")
	}
}

func (h *AdminHandler) ListResources(c *gin.Context) {
	var r req.AdminResourceListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListResources(r.Status, r.Keyword, r.ResourceType, r.CourseID, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) DeleteResource(c *gin.Context) {
	resourceID, ok := parsePositiveID(c)
	if !ok {
		return
	}

	var r req.AdminResourceDeleteReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	err := h.adminSvc.DeleteResource(currentUserID(c), resourceID, r.Reason, parseIP(c.ClientIP()))
	switch {
	case errors.Is(err, service.ErrAdminTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "资源不存在")
	case errors.Is(err, service.ErrAdminConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "资源已删除")
	case errors.Is(err, service.ErrAdminInvalidPayload):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "删除原因无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "删除成功")
	}
}

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	var r req.AdminAuditLogListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	items, total, err := h.adminSvc.ListAuditLogs(r.Action, r.OperatorID, r.TargetType, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, gin.H{"items": items, "total": total})
}

func parsePositiveID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return 0, false
	}
	return id, true
}

func currentUserID(c *gin.Context) int64 {
	return c.MustGet(constant.GinUserID).(int64)
}

func parseIP(raw string) net.IP {
	return net.ParseIP(raw)
}

func buildAnnouncementModel(input req.AdminAnnouncementInput) (model.Announcements, error) {
	item := model.Announcements{
		Title:       strings.TrimSpace(input.Title),
		Content:     strings.TrimSpace(input.Content),
		Type:        model.AnnouncementType(defaultString(input.Type, string(model.AnnouncementTypeNotice))),
		IsPinned:    input.IsPinned != nil && *input.IsPinned,
		IsPublished: input.IsPublished == nil || *input.IsPublished,
	}
	if input.PublishedAt != nil {
		parsed, err := parseAdminTime(*input.PublishedAt)
		if err != nil {
			return model.Announcements{}, err
		}
		item.PublishedAt = parsed
	}
	if input.ExpiresAt != nil {
		parsed, err := parseAdminTime(*input.ExpiresAt)
		if err != nil {
			return model.Announcements{}, err
		}
		item.ExpiresAt = parsed
	}
	return item, nil
}

func buildAnnouncementUpdates(input req.AdminAnnouncementInput) (map[string]interface{}, error) {
	updates := map[string]interface{}{}
	if strings.TrimSpace(input.Title) != "" {
		updates["title"] = strings.TrimSpace(input.Title)
	}
	if strings.TrimSpace(input.Content) != "" {
		updates["content"] = strings.TrimSpace(input.Content)
	}
	if input.Type != "" {
		updates["type"] = model.AnnouncementType(input.Type)
	}
	if input.IsPinned != nil {
		updates["is_pinned"] = *input.IsPinned
	}
	if input.IsPublished != nil {
		updates["is_published"] = *input.IsPublished
	}
	if input.PublishedAt != nil {
		parsed, err := parseAdminTime(*input.PublishedAt)
		if err != nil {
			return nil, err
		}
		updates["published_at"] = parsed
	}
	if input.ExpiresAt != nil {
		parsed, err := parseAdminTime(*input.ExpiresAt)
		if err != nil {
			return nil, err
		}
		updates["expires_at"] = parsed
	}
	return updates, nil
}

func derefFloat64(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseAdminTime(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, errInvalidAdminTime
	}
	return parsed, nil
}

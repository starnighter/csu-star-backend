package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TeacherHandler struct {
	teacherSvc *service.TeacherService
}

func NewTeacherHandler(svc *service.TeacherService) *TeacherHandler {
	return &TeacherHandler{teacherSvc: svc}
}

func (h *TeacherHandler) GetTeacherDetail(c *gin.Context) {
	teacherID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || teacherID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var userID int64
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}

	detail, err := h.teacherSvc.GetTeacherDetail(teacherID, userID)
	switch {
	case errors.Is(err, service.ErrTeacherNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, detail)
}

func (h *TeacherHandler) GetSimpleTeachers(c *gin.Context) {
	var r req.TeacherSimpleReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, err := h.teacherSvc.ListSimpleTeachers(r.Q)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items})
}

func (h *TeacherHandler) GetTeacherRankings(c *gin.Context) {
	var r req.TeacherRankingReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, total, err := h.teacherSvc.ListTeacherRankings(repo.TeacherRankingQuery{
		RankType:     r.RankType,
		DepartmentID: r.DepartmentID,
		Page:         r.Page,
		Size:         r.Size,
		IsIncreased:  r.IsIncreased,
	})
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *TeacherHandler) GetTeacherEvaluations(c *gin.Context) {
	teacherID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || teacherID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.TeacherEvaluationListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var userID int64
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}

	items, total, err := h.teacherSvc.ListTeacherEvaluations(repo.TeacherEvaluationQuery{
		TeacherID: teacherID,
		Sort:      r.Sort,
		Page:      r.Page,
		Size:      r.Size,
	}, userID)
	switch {
	case errors.Is(err, service.ErrTeacherNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *TeacherHandler) GetTeacherEvaluation(c *gin.Context) {
	evaluationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || evaluationID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var userID int64
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}

	item, err := h.teacherSvc.GetTeacherEvaluationItem(evaluationID, userID)
	switch {
	case errors.Is(err, service.ErrTeacherEvaluationNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师评价不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, item)
}

func (h *TeacherHandler) GetTeacherEvaluationReply(c *gin.Context) {
	replyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || replyID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	item, err := h.teacherSvc.GetTeacherEvaluationReplyItem(replyID)
	switch {
	case errors.Is(err, service.ErrTeacherEvaluationReplyNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师评价回复不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, item)
}

func (h *TeacherHandler) CreateTeacherEvaluation(c *gin.Context) {
	teacherID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || teacherID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.TeacherEvaluationInputReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	var courseID *int64
	if r.CourseID != "" {
		parsedCourseID, err := parseStringID(r.CourseID)
		if err != nil {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
		courseID = &parsedCourseID
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	evaluation, err := h.teacherSvc.CreateTeacherEvaluation(userID, teacherID, model.TeacherEvaluations{
		CourseID:            courseID,
		TeachingScore:       r.RatingQuality,
		GradingScore:        r.RatingGrading,
		AttendanceScore:     r.RatingAttendance,
		HomeworkScore:       r.RatingHomework,
		GainScore:           r.RatingGain,
		ExamDifficultyScore: r.RatingExamDifficulty,
		Comment:             r.Comment,
		IsAnonymous:         r.IsAnonymous,
	})
	switch {
	case errors.Is(err, service.ErrTeacherNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
		return
	case errors.Is(err, service.ErrCourseNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case errors.Is(err, service.ErrTeacherCourseMismatch):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "该教师未关联此课程")
		return
	case errors.Is(err, service.ErrEvaluationAssociationIncomplete):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "关联课程评价时必须同时填写课程侧 3 个评分维度")
		return
	case errors.Is(err, service.ErrTeacherEvaluationConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "同类型教师评价已存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	item, err := h.teacherSvc.GetTeacherEvaluationItem(evaluation.ID, userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *TeacherHandler) CreateTeacherEvaluationReply(c *gin.Context) {
	evaluationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || evaluationID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.EvaluationReplyInputReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)

	var replyToReplyID int64
	if r.ReplyToReplyID != "" {
		replyToReplyID, err = strconv.ParseInt(r.ReplyToReplyID, 10, 64)
		if err != nil || replyToReplyID <= 0 {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
	}

	var replyToUserID int64
	if r.ReplyToUserID != "" {
		replyToUserID, err = strconv.ParseInt(r.ReplyToUserID, 10, 64)
		if err != nil || replyToUserID <= 0 {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
	}

	reply, err := h.teacherSvc.CreateTeacherEvaluationReply(userID, evaluationID, r.Content, replyToReplyID, replyToUserID, r.IsAnonymous)
	switch {
	case errors.Is(err, service.ErrTeacherEvaluationNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师评价不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	item, err := h.teacherSvc.GetTeacherEvaluationReplyItem(reply.ID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *TeacherHandler) UpdateTeacherEvaluationReply(c *gin.Context) {
	replyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || replyID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	var r req.EvaluationReplyUpdateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	reply, err := h.teacherSvc.UpdateTeacherEvaluationReply(userID, replyID, r.Content, r.IsAnonymous)
	switch {
	case errors.Is(err, service.ErrTeacherEvaluationReplyNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师评价回复不存在")
		return
	case errors.Is(err, service.ErrTeacherEvaluationReplyForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权修改该教师评价回复")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}
	item, err := h.teacherSvc.GetTeacherEvaluationReplyItem(reply.ID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *TeacherHandler) DeleteTeacherEvaluationReply(c *gin.Context) {
	replyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || replyID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	err = h.teacherSvc.DeleteTeacherEvaluationReply(userID, userRole, replyID)
	switch {
	case errors.Is(err, service.ErrTeacherEvaluationReplyNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师评价回复不存在")
		return
	case errors.Is(err, service.ErrTeacherEvaluationReplyForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权删除该教师评价回复")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}
	resp.SuccessMsg(c, "删除教师评价回复成功")
}

func (h *TeacherHandler) UpdateTeacherEvaluation(c *gin.Context) {
	evaluationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || evaluationID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.TeacherEvaluationInputReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	var courseID *int64
	if r.CourseID != "" {
		parsedCourseID, err := parseStringID(r.CourseID)
		if err != nil {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
		courseID = &parsedCourseID
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	_, err = h.teacherSvc.UpdateTeacherEvaluation(userID, userRole, evaluationID, model.TeacherEvaluations{
		CourseID:            courseID,
		TeachingScore:       r.RatingQuality,
		GradingScore:        r.RatingGrading,
		AttendanceScore:     r.RatingAttendance,
		HomeworkScore:       r.RatingHomework,
		GainScore:           r.RatingGain,
		ExamDifficultyScore: r.RatingExamDifficulty,
		Comment:             r.Comment,
		IsAnonymous:         r.IsAnonymous,
	})
	switch {
	case errors.Is(err, service.ErrTeacherEvaluationNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师评价不存在")
		return
	case errors.Is(err, service.ErrTeacherEvaluationForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权修改该教师评价")
		return
	case errors.Is(err, service.ErrCourseNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case errors.Is(err, service.ErrTeacherCourseMismatch):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "该教师未关联此课程")
		return
	case errors.Is(err, service.ErrEvaluationAssociationIncomplete):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "关联课程评价时必须同时填写课程侧 3 个评分维度")
		return
	case errors.Is(err, service.ErrTeacherEvaluationConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "该教师评价与现有记录冲突")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}
	item, err := h.teacherSvc.GetTeacherEvaluationItem(evaluationID, userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *TeacherHandler) DeleteTeacherEvaluation(c *gin.Context) {
	evaluationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || evaluationID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	err = h.teacherSvc.DeleteTeacherEvaluation(userID, userRole, evaluationID)
	switch {
	case errors.Is(err, service.ErrTeacherEvaluationNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师评价不存在")
		return
	case errors.Is(err, service.ErrTeacherEvaluationForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权删除该教师评价")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.SuccessMsg(c, "删除教师评价成功")
}

func (h *TeacherHandler) GetMyTeacherEvaluations(c *gin.Context) {
	var r req.MyTeacherEvaluationsReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	items, total, err := h.teacherSvc.ListMyTeacherEvaluations(userID, r.Page, r.Size)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *TeacherHandler) RandomShowTeachers(c *gin.Context) {
	teachers, err := h.teacherSvc.RandomShowTeachers()
	if err != nil {
		failInternalWithLog(c, err)
	}
	resp.Success(c, teachers)
}

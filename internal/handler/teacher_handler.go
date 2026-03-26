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

func (h *TeacherHandler) GetTeachers(c *gin.Context) {
	var r req.TeacherListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, total, err := h.teacherSvc.ListTeachers(repo.TeacherListQuery{
		Q:            r.Q,
		DepartmentID: r.DepartmentID,
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

func (h *TeacherHandler) GetTeacherDetail(c *gin.Context) {
	teacherID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || teacherID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	detail, err := h.teacherSvc.GetTeacherDetail(teacherID)
	switch {
	case errors.Is(err, service.ErrTeacherNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, detail)
}

func (h *TeacherHandler) GetTeacherRankings(c *gin.Context) {
	var r req.TeacherRankingReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, total, err := h.teacherSvc.ListTeacherRankings(repo.TeacherRankingQuery{
		RankType:     r.RankType,
		Period:       r.Period,
		DepartmentID: r.DepartmentID,
		Page:         r.Page,
		Size:         r.Size,
		IsIncreased:  r.IsIncreased,
	})
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
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
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
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

	userID := c.MustGet(constant.GinUserID).(int64)
	evaluation, err := h.teacherSvc.CreateTeacherEvaluation(userID, teacherID, model.TeacherEvaluations{
		CourseID:        r.CourseID,
		TeachingScore:   r.RatingQuality,
		GradingScore:    r.RatingGrading,
		AttendanceScore: r.RatingAttendance,
		Comment:         r.Comment,
		IsAnonymous:     r.IsAnonymous,
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
	case errors.Is(err, service.ErrTeacherEvaluationConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "你已对该教师的这门课程发表过评价")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, evaluation)
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

	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	evaluation, err := h.teacherSvc.UpdateTeacherEvaluation(userID, userRole, evaluationID, model.TeacherEvaluations{
		CourseID:        r.CourseID,
		TeachingScore:   r.RatingQuality,
		GradingScore:    r.RatingGrading,
		AttendanceScore: r.RatingAttendance,
		Comment:         r.Comment,
		IsAnonymous:     r.IsAnonymous,
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
	case errors.Is(err, service.ErrTeacherEvaluationConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "该评价与现有记录冲突")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, evaluation)
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
		resp.Fail(c, constant.InternalServerErr.Error())
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
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

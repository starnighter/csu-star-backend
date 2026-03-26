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

type CourseHandler struct {
	courseSvc *service.CourseService
}

func NewCourseHandler(svc *service.CourseService) *CourseHandler {
	return &CourseHandler{courseSvc: svc}
}

func (h *CourseHandler) GetCourses(c *gin.Context) {
	var r req.CourseListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, total, err := h.courseSvc.ListCourses(repo.CourseListQuery{
		Q:          r.Q,
		CourseType: r.CourseType,
		Sort:       r.Sort,
		Page:       r.Page,
		Size:       r.Size,
	})
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *CourseHandler) GetCourseDetail(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || courseID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	detail, err := h.courseSvc.GetCourseDetail(courseID)
	switch {
	case errors.Is(err, service.ErrCourseDetailNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, detail)
}

func (h *CourseHandler) GetCourseRankings(c *gin.Context) {
	var r req.CourseRankingReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, total, err := h.courseSvc.ListCourseRankings(repo.CourseRankingQuery{
		RankType:    r.RankType,
		Period:      r.Period,
		Page:        r.Page,
		Size:        r.Size,
		IsIncreased: r.IsIncreased,
	})
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *CourseHandler) GetCourseEvaluations(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || courseID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.CourseEvaluationListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var userID int64
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}

	items, total, err := h.courseSvc.ListCourseEvaluations(repo.CourseEvaluationQuery{
		CourseID: courseID,
		Sort:     r.Sort,
		Page:     r.Page,
		Size:     r.Size,
	}, userID)
	switch {
	case errors.Is(err, service.ErrCourseDetailNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *CourseHandler) CreateCourseEvaluation(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || courseID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.CourseEvaluationInputReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	evaluation, err := h.courseSvc.CreateCourseEvaluation(userID, courseID, model.CourseEvaluations{
		WorkloadScore:   r.RatingHomework,
		GainScore:       r.RatingGain,
		DifficultyScore: r.RatingExamDifficulty,
		Comment:         r.Comment,
		IsAnonymous:     r.IsAnonymous,
	})
	switch {
	case errors.Is(err, service.ErrCourseDetailNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case errors.Is(err, service.ErrCourseEvaluationConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "你已对该课程发表过评价")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, evaluation)
}

func (h *CourseHandler) UpdateCourseEvaluation(c *gin.Context) {
	evaluationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || evaluationID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.CourseEvaluationInputReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	evaluation, err := h.courseSvc.UpdateCourseEvaluation(userID, userRole, evaluationID, model.CourseEvaluations{
		WorkloadScore:   r.RatingHomework,
		GainScore:       r.RatingGain,
		DifficultyScore: r.RatingExamDifficulty,
		Comment:         r.Comment,
		IsAnonymous:     r.IsAnonymous,
	})
	switch {
	case errors.Is(err, service.ErrCourseEvaluationNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程评价不存在")
		return
	case errors.Is(err, service.ErrCourseEvaluationForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权修改该课程评价")
		return
	case errors.Is(err, service.ErrCourseEvaluationConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "该评价与现有记录冲突")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, evaluation)
}

func (h *CourseHandler) DeleteCourseEvaluation(c *gin.Context) {
	evaluationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || evaluationID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	err = h.courseSvc.DeleteCourseEvaluation(userID, userRole, evaluationID)
	switch {
	case errors.Is(err, service.ErrCourseEvaluationNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程评价不存在")
		return
	case errors.Is(err, service.ErrCourseEvaluationForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权删除该课程评价")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.SuccessMsg(c, "删除课程评价成功")
}

func (h *CourseHandler) GetMyCourseEvaluations(c *gin.Context) {
	var r req.MyCourseEvaluationsReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	items, total, err := h.courseSvc.ListMyCourseEvaluations(userID, r.Page, r.Size)
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

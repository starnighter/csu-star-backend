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

func (h *CourseHandler) GetCourseDetail(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || courseID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var userID int64
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}

	detail, err := h.courseSvc.GetCourseDetail(courseID, userID)
	switch {
	case errors.Is(err, service.ErrCourseDetailNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, detail)
}

func (h *CourseHandler) GetSimpleCourses(c *gin.Context) {
	var r req.CourseSimpleReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, err := h.courseSvc.ListSimpleCourses(r.Q)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items})
}

func (h *CourseHandler) CreateCourseTeacherRelation(c *gin.Context) {
	var r req.CourseTeacherRelationCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	courseID, err := parseStringID(r.CourseID)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	teacherID, err := parseStringID(r.TeacherID)
	if err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	relation, err := h.courseSvc.CreateCourseTeacherRelation(courseID, teacherID)
	switch {
	case errors.Is(err, service.ErrCourseDetailNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case errors.Is(err, service.ErrTeacherNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
		return
	case errors.Is(err, service.ErrCourseTeacherRelationConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "该教师与课程已关联")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{
		"course_id":  strconv.FormatInt(relation.CourseID, 10),
		"teacher_id": strconv.FormatInt(relation.TeacherID, 10),
	})
}

func (h *CourseHandler) GetCourseRankings(c *gin.Context) {
	var r req.CourseRankingReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, total, err := h.courseSvc.ListCourseRankings(repo.CourseRankingQuery{
		RankType:    r.RankType,
		Page:        r.Page,
		Size:        r.Size,
		IsIncreased: r.IsIncreased,
	})
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *CourseHandler) GetCourseResourceCollectionDetail(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("course_id"), 10, 64)
	if err != nil || courseID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var r req.CourseResourceCollectionReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	detail, err := h.courseSvc.GetCourseResourceCollectionDetail(repo.CourseResourceCollectionQuery{
		CourseID:     courseID,
		Sort:         r.Sort,
		ResourceType: r.ResourceType,
		Page:         r.Page,
		Size:         r.Size,
	})
	switch {
	case errors.Is(err, service.ErrCourseDetailNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, detail)
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
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *CourseHandler) GetCourseEvaluation(c *gin.Context) {
	evaluationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || evaluationID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	var userID int64
	if v, ok := c.Get(constant.GinUserID); ok {
		userID = v.(int64)
	}

	item, err := h.courseSvc.GetCourseEvaluationItem(evaluationID, userID)
	switch {
	case errors.Is(err, service.ErrCourseEvaluationNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程评价不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, item)
}

func (h *CourseHandler) GetCourseEvaluationReply(c *gin.Context) {
	replyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || replyID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	item, err := h.courseSvc.GetCourseEvaluationReplyItem(replyID)
	switch {
	case errors.Is(err, service.ErrCourseEvaluationReplyNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程评价回复不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, item)
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
	var teacherID *int64
	if r.TeacherID != "" {
		parsedTeacherID, err := parseStringID(r.TeacherID)
		if err != nil {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
		teacherID = &parsedTeacherID
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	evaluation, err := h.courseSvc.CreateCourseEvaluation(userID, courseID, model.CourseEvaluations{
		TeacherID:       teacherID,
		WorkloadScore:   r.RatingHomework,
		GainScore:       r.RatingGain,
		DifficultyScore: r.RatingExamDifficulty,
		TeachingScore:   r.RatingQuality,
		GradingScore:    r.RatingGrading,
		AttendanceScore: r.RatingAttendance,
		Comment:         r.Comment,
		IsAnonymous:     r.IsAnonymous,
	})
	switch {
	case errors.Is(err, service.ErrCourseDetailNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程不存在")
		return
	case errors.Is(err, service.ErrTeacherNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
		return
	case errors.Is(err, service.ErrTeacherCourseMismatch):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "该课程未关联此教师")
		return
	case errors.Is(err, service.ErrEvaluationAssociationIncomplete):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "关联教师评价时必须同时填写教师侧 3 个评分维度")
		return
	case errors.Is(err, service.ErrCourseEvaluationConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "同类型课程评价已存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	item, err := h.courseSvc.GetCourseEvaluationItem(evaluation.ID, userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *CourseHandler) CreateCourseEvaluationReply(c *gin.Context) {
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

	reply, err := h.courseSvc.CreateCourseEvaluationReply(userID, evaluationID, r.Content, replyToReplyID, replyToUserID, r.IsAnonymous)
	switch {
	case errors.Is(err, service.ErrCourseEvaluationNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程评价不存在")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	item, err := h.courseSvc.GetCourseEvaluationReplyItem(reply.ID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *CourseHandler) UpdateCourseEvaluationReply(c *gin.Context) {
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
	reply, err := h.courseSvc.UpdateCourseEvaluationReply(userID, replyID, r.Content, r.IsAnonymous)
	switch {
	case errors.Is(err, service.ErrCourseEvaluationReplyNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程评价回复不存在")
		return
	case errors.Is(err, service.ErrCourseEvaluationReplyForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权修改该课程评价回复")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}
	item, err := h.courseSvc.GetCourseEvaluationReplyItem(reply.ID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
}

func (h *CourseHandler) DeleteCourseEvaluationReply(c *gin.Context) {
	replyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || replyID <= 0 {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	err = h.courseSvc.DeleteCourseEvaluationReply(userID, userRole, replyID)
	switch {
	case errors.Is(err, service.ErrCourseEvaluationReplyNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "课程评价回复不存在")
		return
	case errors.Is(err, service.ErrCourseEvaluationReplyForbidden):
		resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权删除该课程评价回复")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}
	resp.SuccessMsg(c, "删除课程评价回复成功")
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
	var teacherID *int64
	if r.TeacherID != "" {
		parsedTeacherID, err := parseStringID(r.TeacherID)
		if err != nil {
			resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
			return
		}
		teacherID = &parsedTeacherID
	}

	userID := c.MustGet(constant.GinUserID).(int64)
	userRole := c.MustGet(constant.GinUserRole).(string)
	_, err = h.courseSvc.UpdateCourseEvaluation(userID, userRole, evaluationID, model.CourseEvaluations{
		TeacherID:       teacherID,
		WorkloadScore:   r.RatingHomework,
		GainScore:       r.RatingGain,
		DifficultyScore: r.RatingExamDifficulty,
		TeachingScore:   r.RatingQuality,
		GradingScore:    r.RatingGrading,
		AttendanceScore: r.RatingAttendance,
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
	case errors.Is(err, service.ErrTeacherNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "教师不存在")
		return
	case errors.Is(err, service.ErrTeacherCourseMismatch):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "该课程未关联此教师")
		return
	case errors.Is(err, service.ErrEvaluationAssociationIncomplete):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "关联教师评价时必须同时填写教师侧 3 个评分维度")
		return
	case errors.Is(err, service.ErrCourseEvaluationConflict):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "该课程评价与现有记录冲突")
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}
	item, err := h.courseSvc.GetCourseEvaluationItem(evaluationID, userID)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	resp.Success(c, item)
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
		failInternalWithLog(c, err)
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
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, gin.H{"items": items, "total": total})
}

func (h *CourseHandler) RandomShowCourses(c *gin.Context) {
	courses, err := h.courseSvc.RandomShowCourses()
	if err != nil {
		failInternalWithLog(c, err)
	}
	resp.Success(c, courses)
}

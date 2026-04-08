package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RankingHandler struct {
	teacherSvc  *service.TeacherService
	courseSvc   *service.CourseService
	resourceSvc *service.ResourceService
}

func NewRankingHandler(teacherSvc *service.TeacherService, courseSvc *service.CourseService, resourceSvc *service.ResourceService) *RankingHandler {
	return &RankingHandler{
		teacherSvc:  teacherSvc,
		courseSvc:   courseSvc,
		resourceSvc: resourceSvc,
	}
}

func (h *RankingHandler) GetTeacherRankings(c *gin.Context) {
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

func (h *RankingHandler) GetCourseRankings(c *gin.Context) {
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

func (h *RankingHandler) GetResourceRankings(c *gin.Context) {
	var r req.ResourceRankingReq
	if err := c.ShouldBindQuery(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	items, total, err := h.resourceSvc.ListResourceRankings(repo.ResourceRankingQuery{
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

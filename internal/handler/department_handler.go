package handler

import (
	"csu-star-backend/internal/service"
	"net/http"

	"csu-star-backend/internal/resp"

	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	departmentService *service.DepartmentService
}

func NewDepartmentHandler(svc *service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{svc}
}

func (h *DepartmentHandler) GetAllDepartments(c *gin.Context) {
	departments, err := h.departmentService.GetAllDepartments()
	if err != nil {
		resp.FailWithCode(c, http.StatusInternalServerError, resp.CodeFail, err.Error())
		return
	}

	resp.Success(c, departments)
}

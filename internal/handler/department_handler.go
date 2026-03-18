package handler

import (
	"csu-star-backend/internal/service"
	"log"
	"net/http"

	"csu-star-backend/internal/resp"

	"github.com/gin-gonic/gin"
)

type DepartmentHandler interface {
	GetAllDepartments(c *gin.Context)
}

type departmentHandler struct {
	departmentService service.DepartmentService
}

func NewDepartmentHandler(svc service.DepartmentService) DepartmentHandler {
	return &departmentHandler{svc}
}

func (h *departmentHandler) GetAllDepartments(c *gin.Context) {
	departments, err := h.departmentService.GetAllDepartments()
	if err != nil {
		log.Printf("GetAllDepartments failed: %v", err)
		resp.FailWithCode(c, http.StatusInternalServerError, resp.CodeFail, "服务器内部错误")
		return
	}

	resp.Success(c, departments)
}

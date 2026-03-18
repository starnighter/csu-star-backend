package router

import (
	"csu-star-backend/internal/handler"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetUpRouter(db *gorm.DB, client *http.Client) *gin.Engine {
	r := gin.Default()

	departmentRepo := repo.NewDepartmentRepository(db)
	departmentSvc := service.NewDepartmentService(departmentRepo)
	departmentHandler := handler.NewDepartmentHandler(departmentSvc)
	r.GET("/departments", departmentHandler.GetAllDepartments)

	return r
}

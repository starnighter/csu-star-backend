package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpDeptRouter(r *gin.Engine, deptHandler *handler.DepartmentHandler) {
	g := r.Group("/departments")
	g.Use(middlewarepackage.JWTAuth())
	{
		g.GET("/", deptHandler.GetAllDepartments)
	}
}

package router

import (
	"csu-star-backend/internal/handler"
	"csu-star-backend/internal/middleware"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetUpRouter(db *gorm.DB, client *http.Client) *gin.Engine {
	r := gin.Default()

	// 初始化repo
	userRepo := repo.NewUserRepository(db)
	departmentRepo := repo.NewDepartmentRepository(db)
	invitationRepo := repo.NewInvitationRepository(db)

	// 初始化service
	authSvc := service.NewAuthService(userRepo, invitationRepo)
	oauthSvc := service.NewOauthService(userRepo, client)
	departmentSvc := service.NewDepartmentService(departmentRepo)

	// 初始化handler
	authHandler := handler.NewAuthHandler(authSvc, oauthSvc)
	departmentHandler := handler.NewDepartmentHandler(departmentSvc)

	// 公有API无需登录
	publicApi := r.Group("/auth")
	{
		publicApi.POST("/register", authHandler.Register)
	}

	// 受保护的API需要登陆
	protectedApi := r.Group("")
	protectedApi.Use(middlewarepackage.JWTAuth())
	{
		deptApi := protectedApi.Group("/departments")
		{
			deptApi.GET("/", departmentHandler.GetAllDepartments)
		}
	}

	return r
}

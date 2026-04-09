package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/service"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetUpRouter(db *gorm.DB, client *http.Client, trustedProxies []string) (*gin.Engine, error) {
	r := gin.Default()
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}
	r.Use(middlewarepackage.CORS())

	// 初始化repo
	userRepo := repo.NewUserRepository(db)
	departmentRepo := repo.NewDepartmentRepository(db)
	invitationRepo := repo.NewInvitationRepository(db)
	teacherRepo := repo.NewTeacherRepository(db)
	courseRepo := repo.NewCourseRepository(db)
	resourceRepo := repo.NewResourceRepository(db)
	commentRepo := repo.NewCommentRepository(db)
	socialRepo := repo.NewSocialRepository(db)
	miscRepo := repo.NewMiscRepository(db)
	adminRepo := repo.NewAdminRepository(db)

	// 初始化service
	securitySvc := service.NewSecurityService(db)
	authSvc := service.NewAuthService(userRepo, invitationRepo)
	oauthSvc := service.NewOauthService(userRepo, client)
	departmentSvc := service.NewDepartmentService(departmentRepo)
	teacherSvc := service.NewTeacherService(db, teacherRepo, courseRepo, socialRepo)
	courseSvc := service.NewCourseService(db, courseRepo, teacherRepo, socialRepo)
	resourceSvc := service.NewResourceService(db, resourceRepo, courseRepo, socialRepo)
	commentSvc := service.NewCommentService(commentRepo, teacherRepo, courseRepo, resourceRepo, socialRepo)
	socialSvc := service.NewSocialService(db, socialRepo, courseRepo, teacherRepo, resourceRepo, commentRepo)
	miscSvc := service.NewMiscService(db, miscRepo, socialRepo, invitationRepo)
	adminSvc := service.NewAdminService(db, adminRepo, courseRepo, teacherRepo, commentRepo, socialRepo, resourceRepo)
	authSvc.SetSecurityService(securitySvc)
	oauthSvc.SetSecurityService(securitySvc)
	resourceSvc.SetSecurityService(securitySvc)
	middlewarepackage.InitSecurityService(securitySvc)

	// 初始化handler
	authHandler := handler.NewAuthHandler(authSvc, oauthSvc)
	departmentHandler := handler.NewDepartmentHandler(departmentSvc)
	teacherHandler := handler.NewTeacherHandler(teacherSvc)
	courseHandler := handler.NewCourseHandler(courseSvc)
	resourceHandler := handler.NewResourceHandler(resourceSvc)
	rankingHandler := handler.NewRankingHandler(teacherSvc, courseSvc, resourceSvc)
	commentHandler := handler.NewCommentHandler(commentSvc)
	socialHandler := handler.NewSocialHandler(socialSvc)
	miscHandler := handler.NewMiscHandler(miscSvc)
	adminHandler := handler.NewAdminHandler(adminSvc)

	SetupAuthRouter(r, authHandler)
	SetUpDeptRouter(r, departmentHandler)
	SetUpTeacherRouter(r, teacherHandler)
	SetUpCourseRouter(r, courseHandler)
	SetUpResourceRouter(r, resourceHandler)
	SetUpRankingRouter(r, rankingHandler)
	SetUpCommentRouter(r, commentHandler)
	SetUpSocialRouter(r, socialHandler)
	SetUpMiscRouter(r, miscHandler)
	SetUpAdminRouter(r, adminHandler)

	return r, nil
}

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

	// 初始化service
	authSvc := service.NewAuthService(userRepo, invitationRepo)
	oauthSvc := service.NewOauthService(userRepo, client)
	departmentSvc := service.NewDepartmentService(departmentRepo)
	teacherSvc := service.NewTeacherService(teacherRepo, socialRepo)
	courseSvc := service.NewCourseService(courseRepo, socialRepo)
	resourceSvc := service.NewResourceService(resourceRepo, courseRepo, socialRepo)
	commentSvc := service.NewCommentService(commentRepo, teacherRepo, courseRepo, resourceRepo, socialRepo)
	socialSvc := service.NewSocialService(socialRepo)
	miscSvc := service.NewMiscService(miscRepo, socialRepo)

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

	SetupAuthRouter(r, authHandler)
	SetUpDeptRouter(r, departmentHandler)
	SetUpTeacherRouter(r, teacherHandler)
	SetUpCourseRouter(r, courseHandler)
	SetUpResourceRouter(r, resourceHandler)
	SetUpRankingRouter(r, rankingHandler)
	SetUpCommentRouter(r, commentHandler)
	SetUpSocialRouter(r, socialHandler)
	SetUpMiscRouter(r, miscHandler)

	return r
}

package router

import (
	"csu-star-backend/internal/docengine"
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"
	"csu-star-backend/internal/realtime"
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
	mailProviderRepo := repo.NewMailProviderRepository(db)
	wikiRepo := repo.NewWikiRepository(db)

	// compass / 知识广场 document store (local GORM engine; Docmost-swappable via docengine.Store)
	compassStore := docengine.NewGormStore(db)
	if err := compassStore.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("compass auto migrate: %w", err)
	}
	// Repair broken first import (wiki row IDs reused as page IDs) then re-seed.
	if n, err := docengine.RepairAndImportPublishedWiki(db); err != nil {
		return nil, fmt.Errorf("compass wiki import: %w", err)
	} else {
		_ = n
	}

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
	resourceSvc.SetMiscService(miscSvc)
	teacherSvc.SetMiscService(miscSvc)
	courseSvc.SetMiscService(miscSvc)
	authSvc.SetMiscService(miscSvc)
	mailProviderSvc := service.NewMailProviderService(mailProviderRepo)
	wikiSvc := service.NewWikiService(db, wikiRepo)
	compassSvc := service.NewCompassService(compassStore)
	// 注册后发信通道优先取数据库里管理端配置的，本表为空时回落到 config.yaml
	mailProviderSvc.Install()
	middlewarepackage.InitSecurityService(securitySvc)

	// 实时通知 Hub（进程内；多实例时再加 Redis Pub/Sub）
	wsHub := realtime.Default()
	if wsHub == nil {
		wsHub = realtime.Init()
	}

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
	mailProviderHandler := handler.NewMailProviderHandler(mailProviderSvc)
	wikiHandler := handler.NewWikiHandler(wikiSvc)
	compassHandler := handler.NewCompassHandler(compassSvc)
	wsHandler := handler.NewWSHandler(miscSvc, wsHub)

	SetupAuthRouter(r, authHandler)
	SetUpDeptRouter(r, departmentHandler)
	SetUpTeacherRouter(r, teacherHandler)
	SetUpCourseRouter(r, courseHandler)
	SetUpResourceRouter(r, resourceHandler)
	SetUpRankingRouter(r, rankingHandler)
	SetUpCommentRouter(r, commentHandler)
	SetUpSocialRouter(r, socialHandler)
	SetUpMiscRouter(r, miscHandler)
	SetUpWikiRouter(r, wikiHandler)
	SetUpCompassRouter(r, compassHandler)
	SetUpAdminRouter(r, adminHandler, mailProviderHandler, wikiHandler)
	SetUpWSRouter(r, wsHandler)

	return r, nil
}

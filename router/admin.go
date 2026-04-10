package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"
	"csu-star-backend/internal/model"

	"github.com/gin-gonic/gin"
)

func SetUpAdminRouter(r *gin.Engine, adminHandler *handler.AdminHandler) {
	adminGroup := r.Group("/admin")
	adminGroup.Use(
		middlewarepackage.JWTAuth(),
		middlewarepackage.RequireRoles(string(model.UserRoleAdmin), string(model.UserRoleAuditor)),
	)
	{
		adminGroup.GET("/statistics", adminHandler.GetStatistics)
		adminGroup.GET("/reports", adminHandler.ListReports)
		adminGroup.POST("/reports/:id/handle", adminHandler.HandleReport)
		adminGroup.GET("/corrections", adminHandler.ListCorrections)
		adminGroup.POST("/corrections/:id/handle", adminHandler.HandleCorrection)
		adminGroup.GET("/feedbacks", adminHandler.ListFeedbacks)
		adminGroup.POST("/feedbacks/:id/reply", adminHandler.ReplyFeedback)
		adminGroup.GET("/users", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.ListUsers)
		adminGroup.GET("/users/:id/violations", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.ListUserViolations)
		adminGroup.GET("/announcements", adminHandler.ListAnnouncements)
		adminGroup.GET("/courses", adminHandler.ListCourses)
		adminGroup.GET("/courses/:id/relations", adminHandler.ListCourseRelations)
		adminGroup.GET("/teachers", adminHandler.ListTeachers)
		adminGroup.GET("/teachers/:id/relations", adminHandler.ListTeacherRelations)
		adminGroup.GET("/resources", adminHandler.ListResources)
		adminGroup.GET("/audit-logs", adminHandler.ListAuditLogs)

		adminGroup.POST("/users", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.CreateUser)
		adminGroup.POST("/users/:id/ban", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.BanUser)
		adminGroup.POST("/users/:id/unban", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.UnbanUser)
		adminGroup.POST("/users/:id/points", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.AdjustUserPoints)
		adminGroup.POST("/users/:id/notifications", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.SendUserNotification)
		adminGroup.POST("/announcements", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.CreateAnnouncement)
		adminGroup.PUT("/announcements/:id", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.UpdateAnnouncement)
		adminGroup.DELETE("/announcements/:id", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.DeleteAnnouncement)
		adminGroup.POST("/courses", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.CreateCourse)
		adminGroup.PUT("/courses/:id", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.UpdateCourse)
		adminGroup.DELETE("/courses/:id", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.DeleteCourse)
		adminGroup.POST("/courses/:id/relations", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.AddCourseRelation)
		adminGroup.DELETE("/courses/:id/relations/:teacherId", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.RemoveCourseRelation)
		adminGroup.POST("/teachers", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.CreateTeacher)
		adminGroup.PUT("/teachers/:id", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.UpdateTeacher)
		adminGroup.DELETE("/teachers/:id", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.DeleteTeacher)
		adminGroup.POST("/teachers/:id/relations", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.AddTeacherRelation)
		adminGroup.DELETE("/teachers/:id/relations/:courseId", middlewarepackage.RequireRoles(string(model.UserRoleAdmin)), adminHandler.RemoveTeacherRelation)
		adminGroup.POST("/resources/:id/delete", adminHandler.DeleteResource)
	}
}

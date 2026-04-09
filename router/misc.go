package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"
	"csu-star-backend/internal/model"

	"github.com/gin-gonic/gin"
)

func SetUpMiscRouter(r *gin.Engine, miscHandler *handler.MiscHandler) {
	r.GET("/announcements", miscHandler.GetAnnouncements)
	r.GET("/search", middlewarepackage.OptionalJWTAuth(), miscHandler.Search)
	r.GET("/showcase/stats", miscHandler.GetShowcaseStats)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.GET("/me", miscHandler.GetMe)
		authGroup.PUT("/me", miscHandler.UpdateMe)
		authGroup.POST("/me/checkin", miscHandler.DailyCheckin)
		authGroup.GET("/me/email-status", miscHandler.GetEmailStatus)
		authGroup.GET("/me/invite-code", miscHandler.GetMyInviteCode)
		authGroup.GET("/me/downloads", miscHandler.GetMyDownloads)
		authGroup.GET("/me/favorites", miscHandler.GetMyFavorites)
		authGroup.GET("/me/points", miscHandler.GetMyPoints)
		authGroup.GET("/me/contributions", miscHandler.GetMyContributions)
		authGroup.GET("/me/notifications", miscHandler.GetNotifications)
		authGroup.GET("/me/home-notification-summary", miscHandler.GetHomeNotificationSummary)
		authGroup.GET("/me/notifications/unread-count", miscHandler.GetUnreadCount)
		authGroup.POST("/me/notifications/read-all", miscHandler.MarkAllNotificationsRead)
		authGroup.POST("/feedbacks", miscHandler.CreateFeedback)
		authGroup.POST("/supplement-requests", miscHandler.CreateSupplementRequest)
		authGroup.POST("/reports", miscHandler.CreateReport)
		authGroup.POST("/corrections", miscHandler.CreateCorrection)
		authGroup.PATCH("/notifications/:id/read", miscHandler.MarkNotificationRead)
	}

	adminGroup := r.Group("/admin")
	adminGroup.Use(
		middlewarepackage.JWTAuth(),
		middlewarepackage.RequireRoles(string(model.UserRoleAdmin), string(model.UserRoleAuditor)),
	)
	{
		adminGroup.GET("/supplement-requests", miscHandler.ListSupplementRequests)
		adminGroup.GET("/supplement-requests/:id", miscHandler.GetSupplementRequest)
		adminGroup.POST("/supplement-requests/:id/approve", miscHandler.ApproveSupplementRequest)
		adminGroup.POST("/supplement-requests/:id/reject", miscHandler.RejectSupplementRequest)
	}
}

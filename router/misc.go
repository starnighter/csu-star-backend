package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpMiscRouter(r *gin.Engine, miscHandler *handler.MiscHandler) {
	r.GET("/announcements", miscHandler.GetAnnouncements)
	r.GET("/search", middlewarepackage.OptionalJWTAuth(), miscHandler.Search)
	r.GET("/search/hot", miscHandler.GetHotKeywords)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.GET("/me", miscHandler.GetMe)
		authGroup.PUT("/me", miscHandler.UpdateMe)
		authGroup.POST("/me/checkin", miscHandler.DailyCheckin)
		authGroup.GET("/me/email-status", miscHandler.GetEmailStatus)
		authGroup.GET("/me/downloads", miscHandler.GetMyDownloads)
		authGroup.GET("/me/favorites", miscHandler.GetMyFavorites)
		authGroup.GET("/me/points", miscHandler.GetMyPoints)
		authGroup.GET("/me/notifications", miscHandler.GetNotifications)
		authGroup.GET("/me/notifications/unread-count", miscHandler.GetUnreadCount)
		authGroup.POST("/me/notifications/read-all", miscHandler.MarkAllNotificationsRead)
		authGroup.GET("/search/history", miscHandler.GetSearchHistory)
		authGroup.DELETE("/search/history", miscHandler.ClearSearchHistory)
		authGroup.POST("/feedbacks", miscHandler.CreateFeedback)
		authGroup.POST("/reports", miscHandler.CreateReport)
		authGroup.POST("/corrections", miscHandler.CreateCorrection)
		authGroup.PATCH("/notifications/:id/read", miscHandler.MarkNotificationRead)
	}
}

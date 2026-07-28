package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetUpCompassRouter registers login-gated 知识广场 APIs under /compass.
func SetUpCompassRouter(r *gin.Engine, h *handler.CompassHandler) {
	g := r.Group("/compass")
	g.Use(middlewarepackage.JWTAuth())
	{
		g.GET("/feed", h.GetFeed)
		g.GET("/tree", h.GetTree)

		g.GET("/pages/:id", h.GetPage)
		g.PATCH("/pages/:id", h.UpdatePage)
		g.GET("/pages/:id/history", h.ListHistory)
		g.GET("/pages/:id/comments", h.ListComments)
		g.POST("/pages/:id/comments", h.AddComment)
		g.POST("/pages/:id/edit-requests", h.RequestEdit)

		g.POST("/essays", h.CreateEssay)
		g.POST("/collections", h.CreateCollection)
		g.GET("/collections/:id", h.GetCollection)

		g.POST("/author/apply", h.ApplyAuthor)
		g.GET("/author/me", h.AuthorMe)
		g.GET("/author/applications", h.ListAuthorApplications)
		g.POST("/author/applications/:id/review", h.ReviewAuthorApplication)

		g.POST("/edit-requests/:id/review", h.ReviewEditRequest)

		g.GET("/courses/:courseId/root", h.GetCourseRoot)
	}
}

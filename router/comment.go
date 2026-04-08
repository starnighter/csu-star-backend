package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpCommentRouter(r *gin.Engine, commentHandler *handler.CommentHandler) {
	r.GET("/resources/:id/comments", middlewarepackage.OptionalJWTAuth(), commentHandler.GetResourceComments)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.POST("/resources/:id/comments", middlewarepackage.AuthenticatedRateLimit("comment_write", 20, 60), commentHandler.CreateResourceComment)
		authGroup.PUT("/comments/:id", middlewarepackage.AuthenticatedRateLimit("comment_write", 20, 60), commentHandler.UpdateComment)
		authGroup.DELETE("/comments/:id", middlewarepackage.AuthenticatedRateLimit("comment_write", 20, 60), commentHandler.DeleteComment)
	}
}

package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpCommentRouter(r *gin.Engine, commentHandler *handler.CommentHandler) {
	r.GET("/resources/:id/comments", middlewarepackage.OptionalJWTAuth(), commentHandler.GetResourceComments)
	r.GET("/teachers/:id/comments", middlewarepackage.OptionalJWTAuth(), commentHandler.GetTeacherComments)
	r.GET("/courses/:id/comments", middlewarepackage.OptionalJWTAuth(), commentHandler.GetCourseComments)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.POST("/resources/:id/comments", commentHandler.CreateResourceComment)
		authGroup.POST("/teachers/:id/comments", commentHandler.CreateTeacherComment)
		authGroup.POST("/courses/:id/comments", commentHandler.CreateCourseComment)
		authGroup.DELETE("/comments/:id", commentHandler.DeleteComment)
	}
}

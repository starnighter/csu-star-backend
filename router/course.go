package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpCourseRouter(r *gin.Engine, courseHandler *handler.CourseHandler) {
	r.GET("/courses", courseHandler.GetCourses)
	r.GET("/courses/:id", courseHandler.GetCourseDetail)
	r.GET("/courses/:course_id/evaluations", middlewarepackage.OptionalJWTAuth(), courseHandler.GetCourseEvaluations)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.POST("/courses/:course_id/evaluations", courseHandler.CreateCourseEvaluation)
		authGroup.PUT("/course-evaluations/:id", courseHandler.UpdateCourseEvaluation)
		authGroup.DELETE("/course-evaluations/:id", courseHandler.DeleteCourseEvaluation)
		authGroup.GET("/me/course-evaluations", courseHandler.GetMyCourseEvaluations)
	}
}

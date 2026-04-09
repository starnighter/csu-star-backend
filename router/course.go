package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpCourseRouter(r *gin.Engine, courseHandler *handler.CourseHandler) {
	r.GET("/courses/:id", middlewarepackage.OptionalJWTAuth(), courseHandler.GetCourseDetail)
	r.GET("/courses/simple", courseHandler.GetSimpleCourses)
	r.GET("/courses/evaluations/:id", middlewarepackage.OptionalJWTAuth(), courseHandler.GetCourseEvaluations)
	r.GET("/course-evaluations/:id", middlewarepackage.OptionalJWTAuth(), courseHandler.GetCourseEvaluation)
	r.GET("/course-evaluation-replies/:id", middlewarepackage.OptionalJWTAuth(), courseHandler.GetCourseEvaluationReply)
	r.GET("/course-resource-collections/:course_id", middlewarepackage.OptionalJWTAuth(), courseHandler.GetCourseResourceCollectionDetail)
	r.GET("/courses/random-showcase", courseHandler.RandomShowCourses)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.POST("/course-teacher-relations", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), courseHandler.CreateCourseTeacherRelation)
		authGroup.POST("/courses/evaluations/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), courseHandler.CreateCourseEvaluation)
		authGroup.POST("/course-evaluations/:id/replies", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), courseHandler.CreateCourseEvaluationReply)
		authGroup.PUT("/course-evaluation-replies/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), courseHandler.UpdateCourseEvaluationReply)
		authGroup.DELETE("/course-evaluation-replies/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), courseHandler.DeleteCourseEvaluationReply)
		authGroup.PUT("/course-evaluations/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), courseHandler.UpdateCourseEvaluation)
		authGroup.DELETE("/course-evaluations/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), courseHandler.DeleteCourseEvaluation)
		authGroup.GET("/me/course-evaluations", courseHandler.GetMyCourseEvaluations)
	}
}

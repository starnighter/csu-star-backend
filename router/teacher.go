package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpTeacherRouter(r *gin.Engine, teacherHandler *handler.TeacherHandler) {
	r.GET("/teachers/simple", teacherHandler.GetSimpleTeachers)
	r.GET("/teachers/:id", middlewarepackage.OptionalJWTAuth(), teacherHandler.GetTeacherDetail)
	r.GET("/teachers/evaluations/:id", middlewarepackage.OptionalJWTAuth(), teacherHandler.GetTeacherEvaluations)
	r.GET("/teacher-evaluations/:id", middlewarepackage.OptionalJWTAuth(), teacherHandler.GetTeacherEvaluation)
	r.GET("/teacher-evaluation-replies/:id", middlewarepackage.OptionalJWTAuth(), teacherHandler.GetTeacherEvaluationReply)
	r.GET("/teachers/random-showcase", teacherHandler.RandomShowTeachers)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.POST("/teachers/evaluations/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), teacherHandler.CreateTeacherEvaluation)
		authGroup.POST("/teacher-evaluations/:id/replies", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), teacherHandler.CreateTeacherEvaluationReply)
		authGroup.PUT("/teacher-evaluation-replies/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), teacherHandler.UpdateTeacherEvaluationReply)
		authGroup.DELETE("/teacher-evaluation-replies/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), teacherHandler.DeleteTeacherEvaluationReply)
		authGroup.PUT("/teacher-evaluations/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), teacherHandler.UpdateTeacherEvaluation)
		authGroup.DELETE("/teacher-evaluations/:id", middlewarepackage.AuthenticatedRateLimit("evaluation_write", 20, 60), teacherHandler.DeleteTeacherEvaluation)
		authGroup.GET("/me/teacher-evaluations", teacherHandler.GetMyTeacherEvaluations)
	}
}

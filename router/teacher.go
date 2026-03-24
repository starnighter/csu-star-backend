package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpTeacherRouter(r *gin.Engine, teacherHandler *handler.TeacherHandler) {
	r.GET("/teachers", teacherHandler.GetTeachers)
	r.GET("/teachers/:id", teacherHandler.GetTeacherDetail)
	r.GET("/teachers/:teacher_id/evaluations", middlewarepackage.OptionalJWTAuth(), teacherHandler.GetTeacherEvaluations)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.POST("/teachers/:teacher_id/evaluations", teacherHandler.CreateTeacherEvaluation)
		authGroup.PUT("/teacher-evaluations/:id", teacherHandler.UpdateTeacherEvaluation)
		authGroup.DELETE("/teacher-evaluations/:id", teacherHandler.DeleteTeacherEvaluation)
		authGroup.GET("/me/teacher-evaluations", teacherHandler.GetMyTeacherEvaluations)
	}
}

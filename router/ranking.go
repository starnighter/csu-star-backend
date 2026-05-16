package router

import (
	"csu-star-backend/internal/handler"

	"github.com/gin-gonic/gin"
)

func SetUpRankingRouter(r *gin.Engine, rankingHandler *handler.RankingHandler) {
	r.GET("/rankings/teachers", rankingHandler.GetTeacherRankings)
	r.GET("/rankings/courses", rankingHandler.GetCourseRankings)
	r.GET("/rankings/resources", rankingHandler.GetResourceRankings)
}

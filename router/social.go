package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpSocialRouter(r *gin.Engine, socialHandler *handler.SocialHandler) {
	g := r.Group("")
	g.Use(middlewarepackage.JWTAuth())
	{
		g.POST("/likes", socialHandler.Like)
		g.DELETE("/likes", socialHandler.Unlike)
		g.POST("/favorites", socialHandler.Favorite)
		g.DELETE("/favorites", socialHandler.Unfavorite)
	}
}

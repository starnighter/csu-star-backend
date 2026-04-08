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
		g.POST("/likes", middlewarepackage.AuthenticatedRateLimit("social_write", 30, 60), socialHandler.Like)
		g.DELETE("/likes", middlewarepackage.AuthenticatedRateLimit("social_write", 30, 60), socialHandler.Unlike)
		g.POST("/favorites", middlewarepackage.AuthenticatedRateLimit("social_write", 30, 60), socialHandler.Favorite)
		g.DELETE("/favorites", middlewarepackage.AuthenticatedRateLimit("social_write", 30, 60), socialHandler.Unfavorite)
	}
}

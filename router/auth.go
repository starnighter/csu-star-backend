package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAuthRouter(r *gin.Engine, authHandler *handler.AuthHandler) {
	g := r.Group("/auth")
	{
		g.POST("/forget", authHandler.ForgetPwd)

		authGroup := g.Group("")
		authGroup.Use(middlewarepackage.JWTAuth())
		{
			authGroup.POST("/refresh", authHandler.Refresh)
			authGroup.POST("/logout", authHandler.Logout)
		}

		emailGroup := g.Group("/email")
		{
			emailGroup.POST("/register", authHandler.Register)
			emailGroup.POST("/captcha", authHandler.SendCaptcha)
			emailGroup.POST("/verify", authHandler.VerifyCaptcha)
			emailGroup.POST("/login", authHandler.Login)

			emailAuthGroup := emailGroup.Group("")
			emailAuthGroup.Use(middlewarepackage.JWTAuth())
			{
				emailAuthGroup.POST("/bind", authHandler.BindEmail)
			}
		}

		oauthGroup := g.Group("/oauth")
		{
			oauthGroup.POST("/login", authHandler.OauthLogin)

			oauthAuthGroup := oauthGroup.Group("")
			oauthAuthGroup.Use(middlewarepackage.JWTAuth())
			{
				oauthAuthGroup.POST("/bind", authHandler.OauthBind)
			}
		}
	}
}

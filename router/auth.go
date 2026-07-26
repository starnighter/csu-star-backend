package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAuthRouter(r *gin.Engine, authHandler *handler.AuthHandler) {
	g := r.Group("/auth")
	{
		g.POST("/forget", middlewarepackage.IPBasedRateLimit("auth_captcha_ip", 10, 600), authHandler.ForgetPwd)
		g.POST("/refresh", authHandler.Refresh)

		authGroup := g.Group("")
		authGroup.Use(middlewarepackage.JWTAuth())
		{
			authGroup.POST("/logout", authHandler.Logout)
		}

		emailGroup := g.Group("/email")
		{
			// 注册与 captcha 同档限流：开放域名后防止批量抢注
			emailGroup.POST("/register", middlewarepackage.IPBasedRateLimit("auth_register_ip", 10, 600), authHandler.Register)
			emailGroup.POST("/captcha", middlewarepackage.IPBasedRateLimit("auth_captcha_ip", 10, 600), authHandler.SendCaptcha)
			emailGroup.POST("/verify", middlewarepackage.IPBasedRateLimit("auth_captcha_ip", 20, 600), authHandler.VerifyCaptcha)
			emailGroup.POST("/login", middlewarepackage.IPBasedRateLimit("auth_login_ip", 30, 60), authHandler.Login)

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

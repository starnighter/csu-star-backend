package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpResourceRouter(r *gin.Engine, resourceHandler *handler.ResourceHandler) {
	r.GET("/resources", resourceHandler.GetResources)
	r.GET("/resources/:id", middlewarepackage.OptionalJWTAuth(), resourceHandler.GetResourceDetail)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.POST("/resources", resourceHandler.CreateResource)
		authGroup.POST("/resources/:id/submit", resourceHandler.SubmitResource)
		authGroup.GET("/resources/:id/download", resourceHandler.DownloadResource)
		authGroup.GET("/me/uploads", resourceHandler.GetMyUploads)
	}
}

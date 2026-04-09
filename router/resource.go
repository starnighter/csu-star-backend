package router

import (
	"csu-star-backend/internal/handler"
	middlewarepackage "csu-star-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpResourceRouter(r *gin.Engine, resourceHandler *handler.ResourceHandler) {
	r.GET("/resources/:id", middlewarepackage.OptionalJWTAuth(), resourceHandler.GetResourceDetail)

	authGroup := r.Group("")
	authGroup.Use(middlewarepackage.JWTAuth())
	{
		authGroup.POST("/resources", resourceHandler.CreateResource)
		authGroup.POST("/resources/finalize", resourceHandler.FinalizeResourceUpload)
		authGroup.POST("/resources/abort", resourceHandler.AbortResourceUpload)
		authGroup.PUT("/resources/:id", resourceHandler.UpdateResource)
		authGroup.DELETE("/resources/:id", resourceHandler.DeleteResource)
		authGroup.GET("/resources/:id/download", resourceHandler.DownloadResource)
		authGroup.GET("/me/resources", resourceHandler.GetMyUploads)
	}
}

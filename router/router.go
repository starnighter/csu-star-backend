package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetUpRouter(db *gorm.DB, client *http.Client) *gin.Engine {
	r := gin.Default()
	return r
}

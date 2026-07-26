package router

import (
	"csu-star-backend/internal/handler"

	"github.com/gin-gonic/gin"
)

// SetUpWikiRouter 注册 wiki 公开读接口。
// 这两个接口面向未登录用户,禁止挂任何会返回 401 的中间件
// (frontend 拦截器收到 401 会跳登录页)。
func SetUpWikiRouter(r *gin.Engine, wikiHandler *handler.WikiHandler) {
	r.GET("/wiki/tree", wikiHandler.GetTree)
	r.GET("/wiki/docs/:section/:slug", wikiHandler.GetDoc)
}

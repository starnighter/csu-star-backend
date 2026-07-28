package router

import (
	"csu-star-backend/internal/handler"

	"github.com/gin-gonic/gin"
)

func SetUpWSRouter(r *gin.Engine, wsHandler *handler.WSHandler) {
	// WebSocket endpoints authenticate via ?token= and must not use JWT middleware
	// that writes JSON error bodies before the upgrade handshake.
	r.GET("/ws/notifications", wsHandler.HandleNotifications)
}

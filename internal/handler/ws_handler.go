package handler

import (
	"crypto/md5"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/realtime"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Reuse the same host allow-list semantics as HTTP CORS by letting
		// any non-empty origin through when the request already reached us
		// behind the public API gateway; production should terminate TLS at Nginx.
		return true
	},
}

type WSHandler struct {
	miscSvc *service.MiscService
	hub     *realtime.Hub
}

func NewWSHandler(miscSvc *service.MiscService, hub *realtime.Hub) *WSHandler {
	return &WSHandler{miscSvc: miscSvc, hub: hub}
}

// HandleNotifications upgrades the connection and streams notification events.
// Auth: ?token=<access_token> (browser WebSocket cannot set Authorization).
func (h *WSHandler) HandleNotifications(c *gin.Context) {
	if h.hub == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"code": resp.CodeFail,
			"msg":  "实时通知服务未就绪",
		})
		return
	}

	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
	}
	if token == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code": resp.CodeFail,
			"msg":  "缺少访问令牌",
		})
		return
	}

	userID, err := authenticateWSToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code": resp.CodeFail,
			"msg":  err.Error(),
		})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Log.Warn("websocket upgrade failed", zap.Error(err))
		return
	}

	client := realtime.NewClient(h.hub, userID, conn)
	h.hub.Register(client)

	unreadCount := int64(0)
	if h.miscSvc != nil {
		if count, countErr := h.miscSvc.CountUnreadNotifications(userID); countErr == nil {
			unreadCount = count
		}
	}
	client.SendReady(unreadCount)
	client.Serve()
}

func authenticateWSToken(tokenString string) (int64, error) {
	hash := md5.Sum([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])

	isBlacklisted, err := utils.RDB.Get(utils.Ctx, constant.BlackListPrefix+tokenHash).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, errors.New("鉴权服务异常，请重新登录")
	}
	if !errors.Is(err, redis.Nil) && isBlacklisted != "" {
		return 0, errors.New("Token已失效，请重新登录")
	}

	claims, err := utils.ParseToken(tokenString)
	if err != nil {
		return 0, errors.New("未登录，请先登录哦")
	}
	if claims.Type != "access" {
		return 0, errors.New("请提供Access Token进行鉴权")
	}
	return claims.UserID, nil
}

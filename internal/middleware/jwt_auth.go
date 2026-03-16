package middlewarepackage

import (
	"crypto/md5"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/resp"
	"csu-star-backend/pkg/utils"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请求头中未提供有效的 Bearer Token"})
			c.Abort()
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// md5哈希加密
		data := []byte(tokenString)
		hash := md5.Sum(data)
		tokenHash := hex.EncodeToString(hash[:])
		// 校验黑名单
		isBlacklisted, err := utils.RDB.Get(utils.Ctx, constant.BlackListPrefix+tokenHash).Result()
		if !errors.Is(err, redis.Nil) && isBlacklisted != "" {
			resp.FailWithCode(c, http.StatusUnauthorized, resp.CodeFail, "Token已失效，请重新登录")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		if claims.Type != "access" {
			resp.FailWithCode(c, http.StatusUnauthorized, resp.CodeFail, "请提供Access Token进行鉴权")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

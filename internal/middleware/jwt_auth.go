package middlewarepackage

import (
	"crypto/md5"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"csu-star-backend/pkg/utils"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type authContext struct {
	tokenHash string
	userID    int64
	userRole  string
}

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := authenticateRequest(c, true)
		if !ok {
			return
		}
		setAuthContext(c, ctx)
		c.Next()
	}
}

func OptionalJWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := authenticateRequest(c, false)
		if !ok {
			return
		}
		if ctx != nil {
			setAuthContext(c, ctx)
		}
		c.Next()
	}
}

func authenticateRequest(c *gin.Context, required bool) (*authContext, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		if !required {
			return nil, true
		}
		resp.FailWithCode(c, http.StatusUnauthorized, resp.CodeFail, "请求头中未提供有效的 Bearer Token")
		c.Abort()
		return nil, false
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	hash := md5.Sum([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])

	isBlacklisted, err := utils.RDB.Get(utils.Ctx, constant.BlackListPrefix+tokenHash).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		resp.FailWithCode(c, http.StatusUnauthorized, resp.CodeFail, "鉴权服务异常，请重新登录")
		c.Abort()
		return nil, false
	}
	if !errors.Is(err, redis.Nil) && isBlacklisted != "" {
		resp.FailWithCode(c, http.StatusUnauthorized, resp.CodeFail, "Token已失效，请重新登录")
		c.Abort()
		return nil, false
	}

	claims, err := utils.ParseToken(tokenString)
	if err != nil {
		resp.FailWithCode(c, http.StatusUnauthorized, resp.CodeFail, "未登录，请先登录哦")
		c.Abort()
		return nil, false
	}
	if claims.Type != "access" {
		resp.FailWithCode(c, http.StatusUnauthorized, resp.CodeFail, "请提供Access Token进行鉴权")
		c.Abort()
		return nil, false
	}

	if securitySvc != nil {
		_, banDecision, err := securitySvc.EnforceUserAccess(claims.UserID)
		if err != nil {
			if errors.Is(err, service.ErrSecurityUserBanned) {
				code := constant.UserBannedErr.Code
				msg := constant.UserBannedErr.Msg
				if banDecision != nil && banDecision.BanSource == model.UserBanSourceSystem {
					code = constant.UserAutoBannedErr.Code
					msg = constant.UserAutoBannedErr.Msg
				}
				resp.FailWithData(c, http.StatusForbidden, code, msg, service.BuildRiskData(banDecision))
				c.Abort()
				return nil, false
			}
			resp.Fail(c, constant.InternalServerErr.Error())
			c.Abort()
			return nil, false
		}
	}

	return &authContext{
		tokenHash: tokenHash,
		userID:    claims.UserID,
		userRole:  claims.UserRole,
	}, true
}

func setAuthContext(c *gin.Context, ctx *authContext) {
	c.Set(constant.GinAccessTokenHash, ctx.tokenHash)
	c.Set(constant.GinUserID, ctx.userID)
	c.Set(constant.GinUserRole, ctx.userRole)
}

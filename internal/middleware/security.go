package middlewarepackage

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"csu-star-backend/pkg/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var securitySvc *service.SecurityService

func InitSecurityService(svc *service.SecurityService) {
	securitySvc = svc
}

func AuthenticatedRateLimit(scope string, limit int64, windowSeconds int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDValue, ok := c.Get(constant.GinUserID)
		if !ok {
			c.Next()
			return
		}
		userID := userIDValue.(int64)
		key := service.BuildRateLimitKey(scope, "user", strconv.FormatInt(userID, 10))
		decision, err := securitySvc.RateLimit(c.Request.Context(), utils.RDB, key, limit, time.Duration(windowSeconds)*time.Second)
		if err != nil {
			resp.Fail(c, constant.InternalServerErr.Error())
			c.Abort()
			return
		}
		if decision.Allowed {
			c.Next()
			return
		}

		resp.FailWithData(c, http.StatusTooManyRequests, constant.TooManyRequestsErr.Code, constant.TooManyRequestsErr.Msg, service.BuildRateLimitData(decision.RetryAfter, scope))
		c.Abort()
	}
}

func IPBasedRateLimit(scope string, limit int64, windowSeconds int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := service.BuildRateLimitKey(scope, "ip", c.ClientIP())
		decision, err := securitySvc.RateLimit(c.Request.Context(), utils.RDB, key, limit, time.Duration(windowSeconds)*time.Second)
		if err != nil {
			resp.Fail(c, constant.InternalServerErr.Error())
			c.Abort()
			return
		}
		if decision.Allowed {
			c.Next()
			return
		}
		resp.FailWithData(c, http.StatusTooManyRequests, constant.TooManyRequestsErr.Code, constant.TooManyRequestsErr.Msg, service.BuildRateLimitData(decision.RetryAfter, scope))
		c.Abort()
	}
}

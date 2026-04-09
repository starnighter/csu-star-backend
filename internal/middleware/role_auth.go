package middlewarepackage

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/resp"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		roleValue, ok := c.Get(constant.GinUserRole)
		role, valid := roleValue.(string)
		if !ok || !valid {
			resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权限访问")
			c.Abort()
			return
		}

		if _, exists := allowed[role]; !exists {
			resp.FailWithCode(c, http.StatusForbidden, resp.CodeFail, "无权限访问")
			c.Abort()
			return
		}

		c.Next()
	}
}

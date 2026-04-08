package middlewarepackage

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

var allowedCORSHosts = map[string]struct{}{
	"localhost":          {},
	"127.0.0.1":          {},
	"::1":                {},
	"admin.csustar.wiki": {},
	"csustar.wiki":       {},
	"www.csustar.wiki":   {},
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if isAllowedOrigin(origin) {
			headers := c.Writer.Header()
			headers.Set("Access-Control-Allow-Origin", origin)
			headers.Set("Access-Control-Allow-Credentials", "true")
			headers.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Content-Length, Accept, Origin, X-Requested-With")
			headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			headers.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
			headers.Set("Access-Control-Max-Age", "86400")
			headers.Add("Vary", "Origin")
			headers.Add("Vary", "Access-Control-Request-Method")
			headers.Add("Vary", "Access-Control-Request-Headers")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isAllowedOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	host := parsed.Hostname()
	if host == "" {
		return false
	}

	host = strings.ToLower(host)
	_, ok := allowedCORSHosts[host]
	return ok
}

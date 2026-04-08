package router

import (
	"csu-star-backend/internal/handler"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetUpAdminRouterDoesNotRegisterResourceRestore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	SetUpAdminRouter(r, &handler.AdminHandler{})

	for _, route := range r.Routes() {
		if route.Method == http.MethodPost && route.Path == "/admin/resources/:id/restore" {
			t.Fatalf("unexpected restore route registered")
		}
	}
}

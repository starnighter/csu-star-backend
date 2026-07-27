package router

import (
	"csu-star-backend/internal/docengine"
	"csu-star-backend/internal/handler"
	"csu-star-backend/internal/service"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCompassRoutesRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewCompassHandler(service.NewCompassService(docengine.NewMemoryStore()))
	SetUpCompassRouter(r, h)

	paths := []string{
		"/compass/feed",
		"/compass/pages/1",
		"/compass/pages/1/history",
		"/compass/courses/9/root",
		"/compass/tree",
	}
	for _, p := range paths {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, p, nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s want 401 got %d body=%s", p, w.Code, w.Body.String())
		}
	}
}

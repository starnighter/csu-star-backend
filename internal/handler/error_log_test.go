package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/resp"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"csu-star-backend/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestFailInternalWithLogWritesLogAndResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, observed := observer.New(zap.ErrorLevel)
	previous := logger.Log
	logger.Log = zap.New(core)
	defer func() {
		logger.Log = previous
	}()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/test/internal-error", nil)
	ctx.Set(constant.GinUserID, int64(7))

	failInternalWithLog(ctx, errors.New("boom"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response resp.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid response json, got %v", err)
	}
	if response.Msg != constant.InternalServerErr.Error() {
		t.Fatalf("expected response msg %q, got %q", constant.InternalServerErr.Error(), response.Msg)
	}

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	fields := entries[0].ContextMap()
	if fields["method"] != http.MethodGet {
		t.Fatalf("expected method GET, got %#v", fields["method"])
	}
	if fields["path"] != "/test/internal-error" {
		t.Fatalf("expected path /test/internal-error, got %#v", fields["path"])
	}
	if fields["user_id"] != int64(7) {
		t.Fatalf("expected user_id 7, got %#v", fields["user_id"])
	}
	if fields["operation"] != "TestFailInternalWithLogWritesLogAndResponse" {
		t.Fatalf("expected operation test name, got %#v", fields["operation"])
	}
}

func TestFailInternalWithLogDoesNotPanicWithoutLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previous := logger.Log
	logger.Log = nil
	defer func() {
		logger.Log = previous
	}()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/test/no-logger", nil)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("expected no panic, got %v", recovered)
		}
	}()

	failInternalWithLog(ctx, errors.New("boom"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestFailInternalWithLogDoesNotPanicWithoutRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, _ := observer.New(zap.ErrorLevel)
	previous := logger.Log
	logger.Log = zap.New(core)
	defer func() {
		logger.Log = previous
	}()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("expected no panic, got %v", recovered)
		}
	}()

	failInternalWithLog(ctx, errors.New("boom"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

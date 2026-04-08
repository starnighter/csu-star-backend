package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/resp"
	"csu-star-backend/logger"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func failInternalWithLog(c *gin.Context, err error, fields ...zap.Field) {
	logHandlerError(c, err, fields...)
	resp.Fail(c, constant.InternalServerErr.Error())
}

func failWithCodeAndLog(c *gin.Context, err error, httpStatus int, code int, msg string, fields ...zap.Field) {
	logHandlerError(c, err, fields...)
	resp.FailWithCode(c, httpStatus, code, msg)
}

func logHandlerError(c *gin.Context, err error, fields ...zap.Field) {
	logFields := []zap.Field{
		zap.String("operation", handlerOperationName()),
		zap.String("method", handlerRequestMethod(c)),
		zap.String("path", handlerRequestPath(c)),
		zap.Error(err),
	}
	if userID, ok := handlerUserID(c); ok {
		logFields = append(logFields, zap.Int64("user_id", userID))
	}
	logFields = append(logFields, fields...)
	if logger.Log == nil {
		return
	}
	logger.Log.Error("handler operation failed", logFields...)
}

func handlerUserID(c *gin.Context) (int64, bool) {
	if c == nil {
		return 0, false
	}
	value, ok := c.Get(constant.GinUserID)
	if !ok {
		return 0, false
	}
	userID, ok := value.(int64)
	return userID, ok
}

func handlerRequestMethod(c *gin.Context) string {
	if c != nil && c.Request != nil {
		return c.Request.Method
	}
	return ""
}

func handlerRequestPath(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if path := c.FullPath(); path != "" {
		return path
	}
	if c.Request != nil && c.Request.URL != nil {
		return c.Request.URL.Path
	}
	return ""
}

func handlerOperationName() string {
	pcs := make([]uintptr, 8)
	count := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:count])
	for {
		frame, more := frames.Next()
		name := strings.TrimPrefix(filepath.Ext(frame.Function), ".")
		if name != "logHandlerError" && name != "failInternalWithLog" && name != "failWithCodeAndLog" && name != "" {
			return name
		}
		if !more {
			break
		}
	}
	return "unknown"
}

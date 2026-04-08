package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一的标准响应结构体
type Response struct {
	Code int         `json:"code"` // 业务状态码
	Msg  string      `json:"msg"`  // 提示信息
	Data interface{} `json:"data"` // 数据载荷
}

const (
	CodeSuccess = 0 // 成功
	CodeFail    = 1 // 失败
)

// Success 返回成功响应 (附带数据)
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  "success",
		Data: data,
	})
}

// SuccessMsg 返回成功响应 (仅提示信息)
func SuccessMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  msg,
		Data: nil,
	})
}

// Fail 常用业务失败返回 (默认 HTTP 状态码 400)
func Fail(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{
		Code: CodeFail,
		Msg:  msg,
		Data: nil,
	})
}

// FailWithCode 自定义错误码和 HTTP 状态码返回
func FailWithCode(c *gin.Context, httpStatus int, code int, msg string) {
	c.JSON(httpStatus, Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}

func FailWithData(c *gin.Context, httpStatus int, code int, msg string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

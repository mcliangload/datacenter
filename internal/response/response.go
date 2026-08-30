package response

import (
	"github.com/gin-gonic/gin"

	"datacenter/internal/errno"
)

// Body 统一响应体
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// OK 成功响应
func OK(c *gin.Context, data interface{}) {
	c.JSON(errno.OK.HTTPStatus, Body{Code: errno.OK.Code, Message: errno.OK.Message, Data: data})
}

// Error 失败响应
func Error(c *gin.Context, err *errno.Error) {
	if err == nil {
		err = errno.ErrInternal
	}
	c.JSON(err.HTTPStatus, Body{Code: err.Code, Message: err.Message})
}

// AbortWithError 中止请求并返回错误（用于中间件）
func AbortWithError(c *gin.Context, err *errno.Error) {
	c.Abort()
	Error(c, err)
}

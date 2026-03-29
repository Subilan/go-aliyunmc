// package h 包含一些通讯过程中常用的数据结构、数据包装函数以及帮助函数
package h

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type httpError struct {
	Code    int
	Message string
}

func (e *httpError) Error() string {
	return e.Message
}

// HttpError 返回一个包装好的带 HTTP 状态码的错误信息的内部表示
func HttpError(code int, message string) error {
	return &httpError{
		Code:    code,
		Message: message,
	}
}

// Details 用于包装错误信息
func Details(reason error) gin.H {
	return gin.H{"details": reason.Error()}
}

// Errorf 用于包装格式化错误信息
func Errorf(format string, args ...any) gin.H {
	return gin.H{"details": fmt.Errorf(format, args...)}
}

// Data 用于包装数据
func Data(data any) gin.H {
	return gin.H{"data": data}
}

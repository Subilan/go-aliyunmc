package events

import "github.com/gin-gonic/gin"

// Error 创建一个有指定详细信息的错误事件
func Error(details string) *Event {
	return Stateless(gin.H{"details": details}, TypeError, false)
}

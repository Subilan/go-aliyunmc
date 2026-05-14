package context_util

import (
	// "github.com/Subilan/go-aliyunmc/session"

	"github.com/Subilan/go-aliyunmc/session"

	"github.com/gin-gonic/gin"
)

// Logout 从当前请求的session中清除用户信息
func Logout(c *gin.Context) {
	session.Remove(c, session.KeyUserId)
	session.Remove(c, session.KeyUsername)
}

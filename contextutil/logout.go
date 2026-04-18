package contextutil

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Logout 从当前请求的session中清除用户信息
func Logout(c *gin.Context) error {
	session := sessions.Default(c)
	session.Clear()

	if err := session.Save(); err != nil {
		return err
	}

	return nil
}
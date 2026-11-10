// Package session 持有 lab 模块独立的登录态。
package session

import (
	"github.com/Subilan/go-aliyunmc/internal/common/session_util"

	"github.com/gin-gonic/gin"
)

const (
	CookieName = "lab_session"
	CookiePath = "/lab-v1"
)

// Manager 是 lab 专属的 session manager。
var Manager = session_util.New(CookieName, CookiePath)

// Logout 清除当前 lab 登录态。
func Logout(c *gin.Context) {
	Manager.Remove(c, session_util.KeyUserId)
	Manager.Remove(c, session_util.KeyUsername)
}

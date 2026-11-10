package user_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/session"

	"github.com/gin-gonic/gin"
)

// HandleLogout 用户登出处理函数
func HandleLogout(c *gin.Context) (any, error) {
	session.Logout(c)
	return nil, nil
}

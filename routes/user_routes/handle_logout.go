package user_routes

import (
	"go-aliyunmc/contextutil"
	"go-aliyunmc/env"
	"go-aliyunmc/logs"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// HandleLogout 用户登出处理函数
func HandleLogout(c *gin.Context) (any, error) {
	// 获取session
	session := sessions.Default(c)

	// 清除session中的用户信息
	session.Clear()

	// 保存session
	if err := session.Save(); err != nil {
		return nil, err
	}

	// 在DEV模式下输出登出信息
	if env.DEV {
		user, _ := contextutil.GetUser(c)
		logs.Dev("用户登出: ID=%d, 用户名=%s", user.ID, user.Username)
	}

	return nil, nil
}

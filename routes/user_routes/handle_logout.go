package user_routes

import (
	"go-aliyunmc/contextutil"
	"go-aliyunmc/env"
	"go-aliyunmc/log_util"

	"github.com/gin-gonic/gin"
)

// HandleLogout 用户登出处理函数
func HandleLogout(c *gin.Context) (any, error) {
	err := contextutil.Logout(c)

	if err != nil {
		return nil, err
	}

	// 在DEV模式下输出登出信息
	if env.DEV {
		user, _ := contextutil.GetUser(c)
		log_util.Debug("用户登出: ID=%d, 用户名=%s", user.ID, user.Username)
	}

	return nil, nil
}

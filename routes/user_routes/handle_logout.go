package user_routes

import (
	"go-aliyunmc/context_util"
	"github.com/gin-gonic/gin"
)

// HandleLogout 用户登出处理函数
func HandleLogout(c *gin.Context) (any, error) {
	context_util.Logout(c)
	return nil, nil
}

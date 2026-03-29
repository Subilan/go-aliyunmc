package user_routes

import (
	"fmt"
	"go-aliyunmc-v2/contextutil"

	"github.com/gin-gonic/gin"
)

// HandleGetProfile 获取用户个人信息
func HandleGetProfile(c *gin.Context) (any, error) {
	// 从context中获取用户信息
	user, exists := contextutil.GetUser(c)
	if !exists {
		return nil, fmt.Errorf("用户信息不存在")
	}
	
	return user, nil
}

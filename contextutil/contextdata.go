package contextutil

import (
	"go-aliyunmc-v2/store/models"

	"github.com/gin-gonic/gin"
)

// GetUser 从gin.Context中获取用户信息
func GetUser(c *gin.Context) (models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return models.User{}, false
	}

	userModel, ok := user.(models.User)
	if !ok {
		return models.User{}, false
	}

	return userModel, true
}

// GetUserID 从gin.Context中获取用户ID
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	id, ok := userID.(uint)
	if !ok {
		return 0, false
	}

	return id, true
}

package contextutil

import (
	"go-aliyunmc/perms"
	"go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
)

const UserKey = "user"
const UserIdKey = "user_id"

// GetUser 从gin.Context中获取用户信息
func GetUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get(UserKey)
	if !exists {
		return &models.User{}, false
	}

	userModel, ok := user.(*models.User)
	if !ok {
		return &models.User{}, false
	}

	return userModel, true
}

// GetUserID 从gin.Context中获取用户ID
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get(UserIdKey)
	if !exists {
		return 0, false
	}

	id, ok := userID.(uint)
	if !ok {
		return 0, false
	}

	return id, true
}

func GetUserRole(c *gin.Context) (perms.Role, bool) {
	user, exists := GetUser(c)
	if !exists {
		return "", false
	}

	return perms.Role(user.Role), true
}
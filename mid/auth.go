package mid

import (
	"go-aliyunmc/env"
	"go-aliyunmc/session"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Auth 认证中间件，用于检查用户是否已登录
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if env.DEV {
			user, err := store.EnsureDevUser()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "DEV用户初始化失败"})
				c.Abort()
				return
			}

			c.Set("user", user)
			c.Set("user_id", user.ID)
			c.Next()
			return
		}

		// 从session中获取用户信息
		userID, exists := session.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		// 根据用户ID获取用户信息
		var user models.User
		if err := store.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		// 将用户信息存储到context中
		c.Set("user", &user)
		c.Set("user_id", userID)

		c.Next()
	}
}

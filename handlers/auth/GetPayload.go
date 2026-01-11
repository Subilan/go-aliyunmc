package auth

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/gin-gonic/gin"
)

func HandleGetPayload() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		userId, _ := c.Get("user_id")
		username, _ := c.Get("username")
		role, _ := c.Get("role")

		return helpers.Data(gin.H{
			"user_id":  userId,
			"username": username,
			"role":     role,
		}), nil
	})
}

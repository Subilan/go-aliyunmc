package auth

import (
	"net/http"

	"github.com/Subilan/gomc-server/helpers"
	"github.com/gin-gonic/gin"
)

func HandleGetPayload() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		userId, exist := c.Get("user_id")

		if !exist {
			return nil, helpers.HttpError{Code: http.StatusUnauthorized, Details: "无法获取用户信息"}
		}

		username, exist := c.Get("username")

		if !exist {
			return nil, helpers.HttpError{Code: http.StatusUnauthorized, Details: "无法获取用户信息"}
		}

		return helpers.Data(gin.H{
			"user_id":  userId,
			"username": username,
		}), nil
	})
}

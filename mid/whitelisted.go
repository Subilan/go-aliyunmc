package mid

import (
	"go-aliyunmc/context_util"
	"go-aliyunmc/env"
	"go-aliyunmc/h"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Whitelisted 中间件用于检查当前用户是否已绑定游戏账号（WhitelistUUID），未绑定则拒绝访问。
func Whitelisted() gin.HandlerFunc {
	return func(c *gin.Context) {
		if env.DEV {
			c.Next()
			return
		}
		
		user, exists := context_util.GetUser(c)

		if !exists {
			c.JSON(http.StatusUnauthorized, h.DetailsF("未登录"))
			c.Abort()
			return
		}

		if user.WhitelistUUID == nil {
			c.JSON(http.StatusForbidden, h.DetailsF("未绑定游戏账号"))
			c.Abort()
			return
		}

		c.Next()
	}
}

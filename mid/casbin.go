package mid

import (
	"go-aliyunmc/casbin"
	"go-aliyunmc/contextutil"
	"go-aliyunmc/h"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Casbin 权限检查中间件
func Casbin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := contextutil.GetUser(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, h.DetailsF("未登录"))
			c.Abort()
			return
		}
		sub := user.Role
		obj := c.Request.URL.Path
		act := c.Request.Method

		allowed, err := casbin.En.Enforce(sub, obj, act)
		if err != nil {
			c.JSON(http.StatusInternalServerError, h.DetailsF("权限检查失败"))
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusForbidden, h.DetailsF("权限不足"))
			c.Abort()
			return
		}

		c.Next()
	}
}

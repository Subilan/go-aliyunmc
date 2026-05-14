package mid

import (
	"github.com/Subilan/go-aliyunmc/context_util"
	"github.com/Subilan/go-aliyunmc/h"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Perm 中间件用于基于 Casbin 规则检查当前用户是否有权访问当前请求路径与方法。
func Perm() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := context_util.GetUserRole(c)

		if !exists {
			c.JSON(http.StatusUnauthorized, h.DetailsF("未登录"))
			c.Abort()
			return
		}

		allowed, err := role.CanRequest(c)

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

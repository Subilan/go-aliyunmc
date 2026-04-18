package mid

import (
	"go-aliyunmc/contextutil"
	"go-aliyunmc/h"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Perm 中间件用于检查访问该路由的用户是否具有>=role的权限等级
func Perm() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := contextutil.GetUserRole(c)

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

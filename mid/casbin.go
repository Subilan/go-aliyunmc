package mid

import (
	"go-aliyunmc/casbin"
	"go-aliyunmc/contextutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Casbin 权限检查中间件
func Casbin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := contextutil.GetUser(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}
		sub := user.Role
		obj := c.Request.URL.Path
		act := c.Request.Method

		allowed, err := casbin.En.Enforce(sub, obj, act)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "权限检查失败"})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			c.Abort()
			return
		}

		c.Next()
	}
}

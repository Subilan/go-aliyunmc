package bot

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/h"

	"github.com/gin-gonic/gin"
)

// 机器人权限范围。scope 之间不做隐式继承。
const (
	ScopeRead    = "read"
	ScopeAccount = "account"
	ScopeAdmin   = "admin"
)

// RequireScope 要求当前机器人拥有指定权限范围。
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := GetPrincipal(c)
		if !ok || !p.HasScope(scope) {
			c.AbortWithStatusJSON(http.StatusForbidden, h.DetailsF("权限不足"))
			return
		}
		c.Next()
	}
}

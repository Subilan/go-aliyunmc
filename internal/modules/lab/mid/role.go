package mid

import (
	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"

	"github.com/gin-gonic/gin"
)

// CurrentRole 返回当前请求用户在 lab 中的角色。
func CurrentRole(c *gin.Context) (perm_util.Role, bool) {
	user, exists := context_util.GetUser(c)
	if !exists {
		return perm_util.RoleBasic, false
	}

	return user.Role, true
}

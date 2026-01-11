package mid

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

func Role(expect consts.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := gctx.ShouldGetUserId(c)

		if err != nil {
			c.JSON(http.StatusUnauthorized, helpers.Details("找不到用户凭据"))
			c.Abort()
			return
		}

		userRole, err := store.GetUserRole(userId, consts.UserRoleUser)

		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.Details("无法获取用户角色"))
			c.Abort()
			return
		}

		if userRole != expect {
			c.JSON(http.StatusForbidden, helpers.Details("拒绝访问"))
			c.Abort()
			return
		}

		c.Next()
	}
}

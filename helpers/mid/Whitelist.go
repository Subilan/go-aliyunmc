package mid

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

func Whitelist() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := gctx.ShouldGetUserId(c)

		if err != nil {
			c.JSON(http.StatusForbidden, helpers.Details(err.Error()))
			c.Abort()
			return
		}

		// 超级用户可跳过白名单检查
		role, _ := store.GetUserRole(userId, consts.UserRoleUser)

		if role >= consts.UserRoleAdmin {
			c.Next()
			return
		}

		bound, exists := store.GetGameBound(userId)

		if !exists {
			c.JSON(http.StatusForbidden, helpers.Details("找不到绑定记录"))
			c.Abort()
			return
		}

		if !bound.Whitelisted {
			c.JSON(http.StatusForbidden, helpers.Details("需要白名单"))
			c.Abort()
			return
		}

		c.Set("game_id", bound.GameId)
		c.Next()
	}
}

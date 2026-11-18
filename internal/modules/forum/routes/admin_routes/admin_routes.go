// Package admin_routes 注册管理后台 API。
package admin_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/auth_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"

	"github.com/gin-gonic/gin"
)

// Bind 注册 /forum-v1/admin 下的全部路由。
func Bind(router *gin.RouterGroup) {
	auth_routes.BindAdmin(router)

	admin := router.Group("")
	admin.Use(mid.AdminAuth())
	admin.Use(mid.RateLimitAdminWrites(routeutil.ForumRate(config.Get().Forum.RateLimit.AdminWritePerMinute)))

	bindDashboard(admin)
	bindAudit(admin)
	bindBoards(admin)
	bindContent(admin)
	bindUsers(admin)
	bindPermissions(admin)
	bindSettings(admin)
	bindUploads(admin)
}

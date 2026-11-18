// Package forum 是论坛业务模块的入口。
package forum

import (
	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/admin_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/auth_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/board_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/notification_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/reply_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/search_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/site_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/thread_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/upload_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/user_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/uploads"

	"github.com/gin-gonic/gin"
)

// Prefix 是 forum API 的模块前缀。
const Prefix = "/forum-v1"

// MustInitialize 初始化 forum 模块独立于 lab 的全局组件。
func MustInitialize() {
	uploads.MustInitialize()
	perms.MustInitialize()
}

// Bind 注册 forum 模块的全部 HTTP 路由。
func Bind(engine *gin.Engine) {
	router := engine.Group(Prefix)

	// 所有写请求先做来源校验与请求体大小限制；GET 与 multipart 上传不受影响。
	router.Use(mid.OriginCheck())
	router.Use(h.BodyLimit(jsonBodyLimit()))

	site_routes.Bind(router)
	auth_routes.Bind(router)
	board_routes.Bind(router)
	thread_routes.Bind(router)
	reply_routes.Bind(router)
	user_routes.Bind(router)
	search_routes.Bind(router)
	notification_routes.Bind(router)
	upload_routes.Bind(router)

	admin_routes.Bind(router.Group("/admin"))
}

// jsonBodyLimit 返回 JSON 写接口的请求体上限。
// JSON 转义最坏情况下每个字符占 6 字节，按主题正文上限估算并留出结构开销。
func jsonBodyLimit() int64 {
	cfg := config.Get().Forum
	limit := int64(cfg.MaxThreadHTMLBytes)*6 + int64(cfg.MaxReplyHTMLBytes) + 64*1024
	if limit < 1<<20 {
		limit = 1 << 20
	}
	return limit
}

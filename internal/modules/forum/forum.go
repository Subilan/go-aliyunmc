// Package forum 是论坛业务模块的入口。当前仅提供模块骨架，尚未实现具体功能。
package forum

import (
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"

	"github.com/gin-gonic/gin"
)

// Prefix 是 forum API 的模块前缀。
const Prefix = "/forum-v1"

// MustInitialize 初始化 forum 模块独立于 lab 的全局组件。
func MustInitialize() {
	perms.MustInitialize()
}

// Bind 注册 forum 模块的 HTTP 路由。
// 论坛功能尚未实现，这里只保留模块路由组，任何请求都会得到 404。
func Bind(engine *gin.Engine) {
	_ = engine.Group(Prefix)
}

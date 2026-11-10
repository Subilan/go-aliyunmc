// Package lab 是 lab 控制台业务模块的入口。
package lab

import (
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/aliyun"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/mc"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/perms"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/remote_util"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/bss_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/changelog_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/instance_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/monitor_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/sample_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/server_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/simple_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/state_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/task_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/routes/user_routes"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/tasks"

	"github.com/gin-gonic/gin"
)

// Prefix 是 lab API 的模块前缀。
const Prefix = "/lab-v1"

// MustInitialize 初始化 lab 模块依赖的全局组件。
func MustInitialize() {
	perms.MustInitialize()
	aliyun.MustInitialize()
	remote_util.MustInitialize()
	tasks.RegisterTaskDefinitions()
	mc.MustLoadData()
}

// Bind 注册 lab 模块的全部 HTTP 路由。
func Bind(engine *gin.Engine) {
	router := engine.Group(Prefix)

	task_routes.Bind(router)
	state_routes.Bind(router)
	monitor_routes.Bind(router)
	user_routes.Bind(router)
	instance_routes.Bind(router)
	server_routes.Bind(router)
	sample_routes.Bind(router)
	bss_routes.Bind(router)
	changelog_routes.Bind(router)
	simple_routes.Bind(router)
}

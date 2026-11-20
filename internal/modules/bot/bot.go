// Package bot 提供内网机器人接入：/bot-v1 只接受来自本机回环地址的请求。
package bot

import (
	"github.com/Subilan/go-aliyunmc/internal/common/config"

	"github.com/gin-gonic/gin"
)

// Prefix 是 bot API 的模块前缀。
const Prefix = "/bot-v1"

// botRuntime 是机器人模块的进程内运行状态。
type botRuntime struct {
	limiter *rateLimiter
}

var globalRuntime *botRuntime

// MustInitialize 初始化机器人模块；未启用时不做任何事。
func MustInitialize() {
	cfg := config.Get().Bot
	if !cfg.Enabled {
		return
	}

	globalRuntime = &botRuntime{
		limiter: newRateLimiter(cfg.RateLimitPerMinute),
	}
}

// Bind 注册机器人路由；未启用时不注册任何路由。
func Bind(engine *gin.Engine) {
	if !config.Get().Bot.Enabled || globalRuntime == nil {
		return
	}

	rt := globalRuntime
	router := engine.Group(Prefix)
	router.Use(RequireLoopback())
	router.Use(RateLimit(rt))
	bindRoutes(router)
}

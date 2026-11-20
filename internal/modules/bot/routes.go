package bot

import (
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"

	"github.com/gin-gonic/gin"
)

// bindRoutes 注册机器人接口。
func bindRoutes(router *gin.RouterGroup) {
	router.GET("/ping", h.G(handlePing))
	router.GET("/server-status", h.G(handleServerStatus))
	// 触发会产生副作用，使用 POST。
	router.POST("/trigger-one-click-start", h.G(handleTriggerOneClickStart))
}

// handlePing 供机器人确认服务可达。
func handlePing(_ *gin.Context) (any, error) {
	return gin.H{
		"ok":   true,
		"time": time.Now().Unix(),
	}, nil
}

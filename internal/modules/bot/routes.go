package bot

import (
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"

	"github.com/gin-gonic/gin"
)

// bindRoutes 注册机器人基础设施路由；业务接口在后续版本加入。
func bindRoutes(router *gin.RouterGroup) {
	router.GET("/ping", h.G(handlePing))
}

// handlePing 供机器人确认服务可达。
func handlePing(_ *gin.Context) (any, error) {
	return gin.H{
		"ok":   true,
		"time": time.Now().Unix(),
	}, nil
}

package bot

import (
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"

	"github.com/gin-gonic/gin"
)

// bindRoutes 注册机器人基础设施路由；业务接口在后续版本加入。
func bindRoutes(router *gin.RouterGroup) {
	router.GET("/ping", h.G(handlePing))
}

// handlePing 供机器人验证密钥、时钟与签名链路。
func handlePing(c *gin.Context) (any, error) {
	p, ok := GetPrincipal(c)
	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "机器人认证失败")
	}

	return gin.H{
		"ok":     true,
		"time":   time.Now().Unix(),
		"key_id": p.KeyID,
	}, nil
}

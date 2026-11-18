// Package site_routes 提供公开站点信息。
package site_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

// Bind 注册站点路由。
func Bind(router *gin.RouterGroup) {
	router.GET("/site", h.G(func(c *gin.Context) (any, error) {
		return service.SiteInfo(), nil
	}))
}

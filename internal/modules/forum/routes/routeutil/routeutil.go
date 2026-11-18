// Package routeutil 提供论坛路由共用的小工具。
package routeutil

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/h"

	"github.com/gin-gonic/gin"
)

// UintParam 解析 uint 路径参数；失败返回 400 错误。
func UintParam(c *gin.Context, name string) (uint, error) {
	value, ok := context_util.ParamUint(c, name)
	if !ok {
		return 0, h.HttpError(http.StatusBadRequest, "无效的 "+name+" 参数")
	}
	return value, nil
}

// ForumRate 返回生效的限流值；全局关闭时返回 0。
func ForumRate(value int) int {
	if !config.Get().Forum.RateLimit.Enabled {
		return 0
	}
	return value
}

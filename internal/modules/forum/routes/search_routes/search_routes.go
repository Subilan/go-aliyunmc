// Package search_routes 提供搜索与热门标签接口。
package search_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type searchQuery struct {
	Query    string `form:"q"`
	Board    string `form:"board"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

type tagQuery struct {
	Board string `form:"board"`
	Limit int    `form:"limit"`
}

// Bind 注册搜索路由。
func Bind(router *gin.RouterGroup) {
	router.GET("/search", mid.OptionalAuth(),
		mid.RateLimitByUser("search", routeutil.ForumRate(config.Get().Forum.RateLimit.SearchPerMinute)),
		mid.OptionalRead(), h.Q(search))
	router.GET("/tags", mid.OptionalAuth(), mid.OptionalRead(), h.Q(hotTags))
}

func search(query searchQuery, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	return service.Search(query.Query, query.Board, query.Page, query.PageSize, user)
}

func hotTags(query tagQuery, c *gin.Context) (any, error) {
	items, err := service.HotTags(query.Board, query.Limit)
	if err != nil {
		return nil, err
	}
	return gin.H{"items": items}, nil
}

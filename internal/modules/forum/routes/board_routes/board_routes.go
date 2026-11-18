// Package board_routes 提供公开板块接口与关注操作。
package board_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type followResult struct {
	Following     bool  `json:"following"`
	FollowerCount int64 `json:"followerCount"`
}

// Bind 注册板块路由。
func Bind(router *gin.RouterGroup) {
	router.GET("/boards", mid.OptionalAuth(), mid.OptionalRead(), h.G(listBoards))
	router.GET("/boards/:slug", mid.OptionalAuth(), mid.OptionalRead(), h.G(getBoard))
	router.PUT("/boards/:slug/follow", mid.Auth(), mid.RequirePermission("board.follow"),
		mid.RateLimitByUser("board-follow", routeutil.ForumRate(config.Get().Forum.RateLimit.LikePerMinute)), h.G(followBoard))
	router.DELETE("/boards/:slug/follow", mid.Auth(), mid.RequirePermission("board.follow"),
		mid.RateLimitByUser("board-follow", routeutil.ForumRate(config.Get().Forum.RateLimit.LikePerMinute)), h.G(unfollowBoard))
}

func listBoards(c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	return service.ListBoards(user)
}

func getBoard(c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	return service.BoardDetail(c.Param("slug"), user)
}

func followBoard(c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	following, count, err := service.FollowBoard(user, c.Param("slug"), true)
	if err != nil {
		return nil, err
	}
	return followResult{Following: following, FollowerCount: count}, nil
}

func unfollowBoard(c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	following, count, err := service.FollowBoard(user, c.Param("slug"), false)
	if err != nil {
		return nil, err
	}
	return followResult{Following: following, FollowerCount: count}, nil
}

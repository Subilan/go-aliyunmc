// Package reply_routes 提供跟帖写操作。
package reply_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type createReplyRequest struct {
	ContentHTML string `json:"contentHtml" binding:"required"`
}

type updateReplyRequest struct {
	ContentHTML string `json:"contentHtml" binding:"required"`
}

// Bind 注册跟帖路由。
func Bind(router *gin.RouterGroup) {
	router.POST("/threads/:id/replies", mid.Auth(), mid.RequirePermission("reply.create"),
		mid.RateLimitByUser("reply-create", routeutil.ForumRate(config.Get().Forum.RateLimit.ReplyPerMinute)), h.B(createReply))
	router.PATCH("/replies/:id", mid.Auth(),
		mid.RateLimitByUser("reply-update", routeutil.ForumRate(config.Get().Forum.RateLimit.EditPerMinute)), h.B(updateReply))
	router.DELETE("/replies/:id", mid.Auth(), h.G(deleteReply))
	router.PUT("/replies/:id/like", mid.Auth(), mid.RequirePermission("like.create"),
		mid.RateLimitByUser("reply-like", routeutil.ForumRate(config.Get().Forum.RateLimit.LikePerMinute)), h.G(likeReply))
	router.DELETE("/replies/:id/like", mid.Auth(), mid.RequirePermission("like.create"),
		mid.RateLimitByUser("reply-like", routeutil.ForumRate(config.Get().Forum.RateLimit.LikePerMinute)), h.G(unlikeReply))
	router.POST("/replies/:id/restore", mid.Auth(), h.G(restoreReply))
}

func createReply(req createReplyRequest, c *gin.Context) (any, error) {
	threadID, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	reply, err := service.CreateReply(user, threadID, req.ContentHTML)
	if err != nil {
		return nil, err
	}
	return service.ReplyDTO(reply, user)
}

func updateReply(req updateReplyRequest, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	reply, err := service.UpdateReply(user, id, req.ContentHTML, false, c)
	if err != nil {
		return nil, err
	}
	return service.ReplyDTO(reply, user)
}

func deleteReply(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	if err := service.DeleteReply(user, id, c.Query("reason"), false, c); err != nil {
		return nil, err
	}
	return nil, nil
}

func likeReply(c *gin.Context) (any, error) {
	return setReplyLike(c, true)
}

func unlikeReply(c *gin.Context) (any, error) {
	return setReplyLike(c, false)
}

func setReplyLike(c *gin.Context, like bool) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	liked, count, err := service.LikeReply(user, id, like)
	if err != nil {
		return nil, err
	}
	return dto.LikeResult{Liked: liked, LikeCount: count}, nil
}

func restoreReply(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	reply, err := service.RestoreReply(user, id, c)
	if err != nil {
		return nil, err
	}
	return service.ReplyDTO(reply, user)
}

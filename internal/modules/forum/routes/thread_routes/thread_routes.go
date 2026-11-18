// Package thread_routes 提供主题的公开读取与写操作。
package thread_routes

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type listQuery struct {
	Board    string `form:"board"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

type recentQuery struct {
	Board string `form:"board"`
	Limit int    `form:"limit"`
}

type createThreadRequest struct {
	BoardID     uint   `json:"boardId" binding:"required"`
	Title       string `json:"title" binding:"required"`
	ContentHTML string `json:"contentHtml" binding:"required"`
}

type updateThreadRequest struct {
	Title       *string `json:"title"`
	ContentHTML *string `json:"contentHtml"`
}

type pinRequest struct {
	Pinned *bool `json:"pinned" binding:"required"`
}

type lockRequest struct {
	Locked *bool `json:"locked" binding:"required"`
}

type moveRequest struct {
	BoardID uint `json:"boardId" binding:"required"`
}

// Bind 注册主题路由。
func Bind(router *gin.RouterGroup) {
	router.GET("/threads", mid.OptionalAuth(), mid.OptionalRead(), h.Q(listThreads))
	router.GET("/pinned-threads", mid.OptionalAuth(), mid.OptionalRead(), h.Q(listPinned))
	router.GET("/threads/:id", mid.OptionalAuth(), mid.OptionalRead(), h.G(getThread))
	router.GET("/threads/:id/replies", mid.OptionalAuth(), mid.OptionalRead(), h.Q(listReplies))
	router.GET("/recent-replies", mid.OptionalAuth(), mid.OptionalRead(), h.Q(recentReplies))

	router.POST("/threads", mid.Auth(), mid.RequirePermission("thread.create"),
		mid.RateLimitByUser("thread-create", routeutil.ForumRate(config.Get().Forum.RateLimit.ThreadPerMinute)), h.B(createThread))
	router.PATCH("/threads/:id", mid.Auth(),
		mid.RateLimitByUser("thread-update", routeutil.ForumRate(config.Get().Forum.RateLimit.EditPerMinute)), h.B(updateThread))
	router.DELETE("/threads/:id", mid.Auth(), h.G(deleteThread))

	router.PUT("/threads/:id/like", mid.Auth(), mid.RequirePermission("like.create"),
		mid.RateLimitByUser("thread-like", routeutil.ForumRate(config.Get().Forum.RateLimit.LikePerMinute)), h.G(likeThread))
	router.DELETE("/threads/:id/like", mid.Auth(), mid.RequirePermission("like.create"),
		mid.RateLimitByUser("thread-like", routeutil.ForumRate(config.Get().Forum.RateLimit.LikePerMinute)), h.G(unlikeThread))

	router.POST("/threads/:id/pin", mid.Auth(), h.B(pinThread))
	router.POST("/threads/:id/lock", mid.Auth(), h.B(lockThread))
	router.POST("/threads/:id/move", mid.Auth(), h.B(moveThread))
	router.POST("/threads/:id/restore", mid.Auth(), h.G(restoreThread))
}

func listThreads(query listQuery, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	return service.ListThreads(query.Board, false, query.Page, query.PageSize, user)
}

func listPinned(query listQuery, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	return service.ListThreads(query.Board, true, query.Page, query.PageSize, user)
}

func getThread(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	return service.ThreadDetail(id, user)
}

func listReplies(query listQuery, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	return service.ThreadReplies(id, query.Page, query.PageSize, user)
}

func recentReplies(query recentQuery, c *gin.Context) (any, error) {
	items, err := service.RecentReplies(query.Board, query.Limit)
	if err != nil {
		return nil, err
	}
	return gin.H{"items": items}, nil
}

func createThread(req createThreadRequest, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	thread, err := service.CreateThread(user, req.BoardID, req.Title, req.ContentHTML)
	if err != nil {
		return nil, err
	}
	return service.ThreadDetail(thread.ID, user)
}

func updateThread(req updateThreadRequest, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	if req.Title == nil && req.ContentHTML == nil {
		return nil, h.HttpError(http.StatusBadRequest, "没有需要修改的字段")
	}

	user, _ := mid.CurrentUser(c)
	thread, err := service.UpdateThread(user, id, req.Title, req.ContentHTML)
	if err != nil {
		return nil, err
	}
	return service.ThreadDetail(thread.ID, user)
}

func deleteThread(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	if err := service.DeleteThread(user, id, c.Query("reason"), false, c); err != nil {
		return nil, err
	}
	return nil, nil
}

func likeThread(c *gin.Context) (any, error) {
	return setThreadLike(c, true)
}

func unlikeThread(c *gin.Context) (any, error) {
	return setThreadLike(c, false)
}

func setThreadLike(c *gin.Context, like bool) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	liked, count, err := service.LikeThread(user, id, like)
	if err != nil {
		return nil, err
	}
	return dto.LikeResult{Liked: liked, LikeCount: count}, nil
}

func pinThread(req pinRequest, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	thread, err := service.AdminUpdateThread(user, id, service.AdminUpdateThreadInput{Pinned: req.Pinned}, c)
	if err != nil {
		return nil, err
	}
	return service.ThreadDetail(thread.ID, user)
}

func lockThread(req lockRequest, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	thread, err := service.AdminUpdateThread(user, id, service.AdminUpdateThreadInput{Locked: req.Locked}, c)
	if err != nil {
		return nil, err
	}
	return service.ThreadDetail(thread.ID, user)
}

func moveThread(req moveRequest, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	thread, err := service.AdminUpdateThread(user, id, service.AdminUpdateThreadInput{BoardID: &req.BoardID}, c)
	if err != nil {
		return nil, err
	}
	return service.ThreadDetail(thread.ID, user)
}

func restoreThread(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	user, _ := mid.CurrentUser(c)
	thread, err := service.RestoreThread(user, id, c)
	if err != nil {
		return nil, err
	}
	return service.ThreadDetail(thread.ID, user)
}

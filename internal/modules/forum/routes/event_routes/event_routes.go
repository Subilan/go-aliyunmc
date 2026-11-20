// Package event_routes 提供论坛前台的统一 SSE 实时事件通道。
package event_routes

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/sse"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/realtime"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type readyPayload struct {
	UnreadCount int64 `json:"unreadCount"`
}

// Bind 注册统一 SSE 路由。一个登录标签页只应建立一条连接。
func Bind(router *gin.RouterGroup) {
	group := router.Group("")
	group.Use(mid.Auth())
	group.GET("/events", stream)
}

func stream(c *gin.Context) {
	user, _ := mid.CurrentUser(c)

	client, err := sse.NewClient(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.DetailsF("无法建立 SSE 连接"))
		return
	}

	hub := realtime.Current()
	if hub == nil {
		sseUnavailable(c)
		return
	}

	ready := sse.Event{
		Event: realtime.EventReady,
		Data:  readyPayload{UnreadCount: service.UnreadCount(user)},
	}
	events, unsubscribe, ok := hub.Subscribe(user.ID, realtime.DefaultBuffer, client.Close, ready)
	if !ok {
		sseUnavailable(c)
		return
	}
	defer unsubscribe()

	client.Attach(events)
	client.Listen()
}

func sseUnavailable(c *gin.Context) {
	// NewClient 已写入 text/event-stream，这里恢复 JSON 响应头。
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusServiceUnavailable, h.DetailsF("实时通道暂不可用"))
}

// Package notification_routes 提供当前用户的通知接口。
package notification_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type listQuery struct {
	Filter   string `form:"filter"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

type markReadRequest struct {
	IDs []uint `json:"ids"`
	All bool   `json:"all"`
}

// Bind 注册通知路由。
func Bind(router *gin.RouterGroup) {
	group := router.Group("")
	group.Use(mid.Auth(), mid.RequirePermission("notification.read_own"))
	{
		group.GET("/notifications", h.Q(list))
		group.GET("/notifications/unread-count", h.G(unreadCount))
		group.POST("/notifications/read", h.B(markRead))
	}
}

func list(query listQuery, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	filter := query.Filter
	if filter != "unread" {
		filter = "all"
	}
	return service.ListNotifications(user, filter, query.Page, query.PageSize)
}

func unreadCount(c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	return gin.H{"count": service.UnreadCount(user)}, nil
}

func markRead(req markReadRequest, c *gin.Context) (any, error) {
	user, _ := mid.CurrentUser(c)
	unread, err := service.MarkNotificationsRead(user, req.IDs, req.All)
	if err != nil {
		return nil, err
	}
	return gin.H{"unreadCount": unread}, nil
}

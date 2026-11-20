// Package realtime 提供论坛前台的单连接实时事件分发。
package realtime

// SSE 事件名。前端在同一个 EventSource 上按名称注册监听。
const (
	EventReady            = "ready"
	EventNotification     = "notification"
	EventNotificationRead = "notification.read"
	EventThreadLike       = "thread.like"
	EventReplyLike        = "reply.like"
	EventUserLike         = "user.like"
	EventBoardFollow      = "board.follow"
)

// DefaultBuffer 是每条 SSE 连接的事件缓冲大小。
const DefaultBuffer = 64

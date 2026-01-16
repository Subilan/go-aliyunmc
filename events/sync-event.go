package events

import "github.com/gin-gonic/gin"

// SyncEventType 表示一个同步事件类型
type SyncEventType string

const (
	// SyncEventClearLastEventId 表示要求前端清楚其存储的 Last-Event-Id
	SyncEventClearLastEventId SyncEventType = "clear_last_event_id"
)

// Sync 创建一个指定类型的同步事件
func Sync(typ SyncEventType) *Event {
	return Stateless(gin.H{"syncType": typ}, TypeSync, false)
}

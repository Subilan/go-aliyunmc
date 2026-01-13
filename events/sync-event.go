package events

import "github.com/gin-gonic/gin"

type SyncEventType string

const (
	SyncEventClearLastEventId SyncEventType = "clear_last_event_id"
)

func Sync(typ SyncEventType) *Event {
	return Stateless(gin.H{"syncType": typ}, TypeSync, false)
}

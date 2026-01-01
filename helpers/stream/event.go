package stream

import (
	"github.com/Subilan/gomc-server/globals"
	"github.com/gin-gonic/gin"
	"go.jetify.com/sse"
)

type Event struct {
	State   *State
	IsError bool
	Content string
}

func (event *Event) Save() error {
	_, err := globals.Pool.Exec("INSERT INTO `pushed_events` (task_id, ord, `type`, is_error, content) VALUES (?, ?, ?, ?, ?)", event.State.TaskId, event.State.Ord, event.State.Type, event.IsError, event.Content)

	if err != nil {
		return err
	}

	return nil
}

type EventErrorType string

const (
	EventErrorInvalidLastEventId EventErrorType = "invalid_last_event_id"
	EventErrorInternal                          = "internal"
)

func ErrorEvent(reason string, typ EventErrorType) *sse.Event {
	return &sse.Event{
		Event: "error",
		Data: gin.H{
			"reason":     reason,
			"error_type": typ,
		},
	}
}

func DataEvent(wrapped Event) *sse.Event {
	return &sse.Event{
		ID: wrapped.State.String(),
		Data: gin.H{
			"type":     wrapped.State.Type,
			"is_error": wrapped.IsError,
			"content":  wrapped.Content,
		},
	}
}

package helpers

import (
	"github.com/gin-gonic/gin"
	"go.jetify.com/sse"
)

func Details(str string) gin.H {
	return gin.H{"details": str}
}

func Data(d any) gin.H {
	return gin.H{"data": d}
}

func PaginData(d any, cnt int, nextToken string) gin.H {
	return gin.H{"data": d, "cnt": cnt, "nextToken": nextToken}
}

type EventErrorType string

const (
	EventErrorInvalidLastEventId EventErrorType = "invalid_last_event_id"
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

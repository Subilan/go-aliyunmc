package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/gin-gonic/gin"
	"go.jetify.com/sse"
)

type EventType int

const (
	TypeDeployment EventType = iota
	TypeServer
	TypeInstance
	TypeError
	TypeSync
)

func Stateless(data any, typ EventType, public bool) *Event {
	marshalledData, _ := json.Marshal(data)
	return &Event{
		EventState: EventState{},
		Type:       typ,
		IsError:    false,
		IsPublic:   public,
		Content:    string(marshalledData),
		CreatedAt:  time.Now(),
	}
}

type Event struct {
	EventState

	Type EventType `json:"type"`

	// IsError 决定前端该信息的显示效果。IsError 并不一定表示此信息与系统的严重错误有关。
	IsError bool `json:"isError"`

	// IsPublic 决定该信息是否可以发送给未登录用户
	IsPublic bool `json:"isPublic"`

	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type EventState struct {
	TaskId *string `json:"taskId"`
	Ord    *int    `json:"ord"`
}

func (s *EventState) String() string {
	if s.TaskId == nil || s.Ord == nil {
		return ""
	}

	return fmt.Sprintf("%s$%d", *s.TaskId, *s.Ord)
}

func (s *EventState) IncrOrd() {
	if s.Ord != nil {
		*(s.Ord)++
	}
}

func StateFromString(stateStr string) (*EventState, error) {
	splitted := strings.Split(stateStr, "$")

	if len(splitted) != 2 {
		return nil, errors.New("invalid state string")
	}

	ord, err := strconv.Atoi(splitted[1])

	if err != nil {
		return nil, err
	}

	return &EventState{
		TaskId: &splitted[0],
		Ord:    &ord,
	}, nil
}

func (event *Event) SSE() *sse.Event {
	return &sse.Event{
		ID: event.EventState.String(),
		Data: gin.H{
			"type":     event.Type,
			"is_error": event.IsError,
			"content":  event.Content,
		},
	}
}

func (event *Event) Insert() error {
	_, err := db.Pool.Exec("INSERT INTO `pushed_events` (task_id, ord, `type`, is_error, is_public, content) VALUES (?, ?, ?, ?, ?, ?)", event.TaskId, event.Ord, event.Type, event.IsError, event.IsPublic, event.Content)

	if err != nil {
		return err
	}

	return nil
}

// Package events 提供与系统事件相关功能。一个系统事件表示系统中发生的一些值得注意的事情，确切地说是一些应当告知用户的事情。
//
// 本系统之所以要引入事件的概念，源于最初考虑的前端页面动态更新功能，该功能让用户能够第一时间看到服务器和实例的状态变化而无需刷新页面。
//
// 该功能有多种实现方式，例如前端向后端接口的轮询（Seatide v1、v2 做法）以及 WebSocket。为了提高程序的可读性并充分发挥 Golang 提供的并发优势，Seatide v3 中不再考虑使用轮询这一方式，而是转向一种按需的设计。
//
// WebSocket 提供了一种“按需”的服务，使得后端可以根据需要向前端推送数据。由于实际上并不需要前端向后端传送数据，为了追求轻量，本系统最终采用的是 Server-Sent Event。此技术亦用于大模型网页对话前端的流式输出内容推送。
//
// 在这样的前提下，系统需要一种能够表示推送的基本单位的数据结构，由此引出系统事件 Event 。
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

// EventType 表示事件的类型
//   - TypeDeployment 未使用。
//   - TypeServer 表示与服务器相关的事件。
//   - TypeInstance 表示与实例相关的事件。
//   - TypeError 表示一个错误事件。该错误由系统提供，属于应用层产生的错误。
//   - TypeSync 表示一个同步事件。这种事件用于告知前端进行一个特定的操作。
type EventType int

const (
	TypeDeployment EventType = iota
	TypeServer
	TypeInstance
	TypeError
	TypeSync
)

// Stateless 返回一个无状态（仅用于传输信息本身）的事件
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

// Event 表示一个事件。
type Event struct {
	// EventState 是该事件的状态
	EventState

	Type EventType `json:"type"`

	// IsError 决定前端该事件的显示效果。IsError 并不一定表示此事件与系统的严重错误有关。
	IsError bool `json:"isError"`

	// IsPublic 决定该事件是否可以发送给未登录用户
	IsPublic bool `json:"isPublic"`

	// Content 是该事件的内容，可能是序列化以后的字符串
	Content string `json:"content"`

	// CreatedAt 是事件的创建时间（由数据库记录）
	CreatedAt time.Time `json:"createdAt"`
}

// EventState 表示事件的状态。
//
// 在当前的设计中，一个有状态的事件必定与一个系统中的任务 store.Task 相关联，因此状态的组成元素包含 TaskId。
type EventState struct {
	// TaskId 是与该事件相关联的任务标识符，并充当 Ord 的命名空间
	TaskId *string `json:"taskId"`

	// Ord 是该事件在该任务内部的顺序（按照时间排序）。顺序的开始是任意的，但不可重复。
	Ord *int `json:"ord"`
}

// String 将事件的状态转换为字符串，是一个简单的序列化过程。
func (s *EventState) String() string {
	if s.TaskId == nil || s.Ord == nil {
		return ""
	}

	return fmt.Sprintf("%s$%d", *s.TaskId, *s.Ord)
}

// IncrOrd 将事件状态中的 Ord 增加 1
func (s *EventState) IncrOrd() {
	if s.Ord != nil {
		*(s.Ord)++
	}
}

// StateFromString 用于从字符串中反序列化出一个事件状态
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

// SSE 将该事件转换为一个 sse.Event 用于发送。
//
// sse.Event 的 Data 的内容虽然是任意的，但为了统一，将其固定为函数体中表示的这种结构。前端应当统一按照此结构进行处理。
//
// sse.Event 的 ID 填入的是事件状态的序列化形式，用于对应 EventSource 的重发机制，也便于前端获取用来进行特殊处理（如本地的持久化）。
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

// Insert 将该 Event 的信息插入到数据库中。
func (event *Event) Insert() error {
	_, err := db.Pool.Exec("INSERT INTO `pushed_events` (task_id, ord, `type`, is_error, is_public, content) VALUES (?, ?, ?, ?, ?, ?)", event.TaskId, event.Ord, event.Type, event.IsError, event.IsPublic, event.Content)

	if err != nil {
		return err
	}

	return nil
}

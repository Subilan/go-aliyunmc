// Package stream 主要包含对服务器推送部分的实现，包括对用户连接的存储和管理、事件的发送等。
//
// “stream”这一名称在本系统中主要用于表达前端开启的用来接收服务器推送，从而保持与服务器的信息同步的长链接。
package stream

import (
	"context"
	"log"
	"sync"

	"github.com/Subilan/go-aliyunmc/events"
	"github.com/google/uuid"
	"go.jetify.com/sse"
)

// UserStream 是已登录用户主动建立的 SSE 连接的封装
type UserStream struct {
	// Id 是该连接的唯一标识
	Id string

	// UserId 是该用户的主键
	UserId int

	// Conn 是 SSE 连接
	Conn *sse.Conn

	// Chan 是推送信息的管道，用于通知用户建立连接时的路由 gorountine 向用户交付信息
	Chan chan *sse.Event

	// Ctx 是该连接的上下文
	Ctx context.Context
}

// userStreams 存储所有用户的连接信息，键为该连接的 UUID，值为连接的 UserStream
var userStreams = make(map[string]*UserStream)

// userStreamsMu 保护 userStreams 的读写
var userStreamsMu sync.Mutex

// taskStreamStates 存储系统中当前正在进行的任务的事件状态信息，键为任务的标识符，值为具体的事件状态信息，用 events.EventState 表示
var taskStreamStates = make(map[string]*events.EventState)

// taskStreamStatesMu 保护 taskStreamStates 的读写
var taskStreamStatesMu sync.Mutex

// Broadcast 向所有已连接的用户传递相同的推送
func Broadcast(wrapped *events.Event) {
	log.Printf("debug: broadcasting event: type: %d, content: %s", wrapped.Type, wrapped.Content)

	if wrapped.IsPublic {
		publicChannel.Publish(wrapped.SSE())
	}

	for _, syncUser := range userStreams {
		syncUser.Send(wrapped)
	}
}

// BroadcastAndSave 向所有已连接的用户推送信息，并保存到数据库中
func BroadcastAndSave(wrapped *events.Event) error {
	err := wrapped.Insert()

	if err != nil {
		return err
	}

	Broadcast(wrapped)

	return nil
}

// Send 向该用户的 Chan 传递一个推送
func (s *UserStream) Send(wrapped *events.Event) {
	s.Chan <- wrapped.SSE()
}

// RecordStateForTask 创建一个全局流状态表项，并将顺序字段设置为 0。
func RecordStateForTask(taskId string) {
	taskStreamStatesMu.Lock()
	defer taskStreamStatesMu.Unlock()

	ord := 0
	taskStreamStates[taskId] = &events.EventState{TaskId: &taskId, Ord: &ord}
}

// DeleteStateOfTask 从全局流状态表中删除 taskId 对应的状态信息。该函数只应该在任务结束后调用。
func DeleteStateOfTask(taskId string) {
	taskStreamStatesMu.Lock()
	defer taskStreamStatesMu.Unlock()
	delete(taskStreamStates, taskId)
}

// GetStateOfTask 从全局流状态表中获取 taskId 对应的状态信息。
//
// Warning: 该方法不会对键的合法性做检查，调用时请确保该键未被删除。
func GetStateOfTask(taskId string) (*events.EventState, bool) {
	taskStreamStatesMu.Lock()
	defer taskStreamStatesMu.Unlock()

	state, ok := taskStreamStates[taskId]

	return state, ok
}

// IncrStateOrdOfTask 为全局流状态表对应状态信息的顺序字段增加 1。
//
// 如果对应的状态信息不存在，则此函数不做操作。
func IncrStateOrdOfTask(taskId string) {
	taskStreamStatesMu.Lock()
	defer taskStreamStatesMu.Unlock()

	if state, ok := taskStreamStates[taskId]; ok && state != nil {
		state.IncrOrd()
	}
}

// Register 将用户的 SSE 连接信息记录到全局表中，用于后续推送信息
func Register(userId int, conn *sse.Conn, ctx context.Context) (string, *UserStream) {
	userStreamsMu.Lock()
	defer userStreamsMu.Unlock()

	connId, _ := uuid.NewRandom()
	connIdStr := connId.String()

	var syncUser = &UserStream{
		Id:     connIdStr,
		UserId: userId,
		Conn:   conn,
		Chan:   make(chan *sse.Event, 16),
		Ctx:    ctx,
	}

	userStreams[connIdStr] = syncUser
	return connIdStr, syncUser
}

// Unregister 从全局表中删除指定用户的 SSE 连接信息
func Unregister(connId string) {
	userStreamsMu.Lock()
	defer userStreamsMu.Unlock()
	delete(userStreams, connId)
}

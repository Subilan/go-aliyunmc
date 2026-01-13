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

var userStreams = make(map[string]*UserStream)
var userStreamsMu sync.Mutex
var globalStreamStates = make(map[string]*events.EventState)
var globalStreamStatesMu sync.Mutex

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
	var ord = 0
	globalStreamStates[taskId] = &events.EventState{TaskId: &taskId, Ord: &ord}
}

// DeleteStateOfTask 从全局流状态表中删除 taskId 对应的状态信息。
// 该函数只应该在任务结束后调用。
func DeleteStateOfTask(taskId string) {
	globalStreamStatesMu.Lock()
	defer globalStreamStatesMu.Unlock()
	delete(globalStreamStates, taskId)
}

// GetStateOfTask 从全局流状态表中获取 taskId 对应的状态信息。
// 该方法不会对键的合法性做检查，调用时请确保该键未被删除。
func GetStateOfTask(taskId string) *events.EventState {
	globalStreamStatesMu.Lock()
	defer globalStreamStatesMu.Unlock()
	return globalStreamStates[taskId]
}

// IncrStateOrdOfTask 为全局流状态表对应状态信息的顺序字段增加 1。
// 该方法不会对键的合法性做检查，调用时请确保该键未被删除。
func IncrStateOrdOfTask(taskId string) {
	globalStreamStatesMu.Lock()
	defer globalStreamStatesMu.Unlock()
	globalStreamStates[taskId].IncrOrd()
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

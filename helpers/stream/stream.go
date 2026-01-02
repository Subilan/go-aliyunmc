package stream

import (
	"context"

	"github.com/Subilan/go-aliyunmc/helpers/store"
	"go.jetify.com/sse"
)

// UserStream 是对用户主动建立的 SSE 连接的封装
type UserStream struct {
	// UserId 是该用户的主键
	UserId int

	// Conn 是 SSE 连接
	Conn *sse.Conn

	// Chan 是推送信息的管道，用于通知用户建立连接时的路由 gorountine 向用户交付信息
	Chan chan *sse.Event

	// Ctx 是该连接的上下文
	Ctx context.Context
}

// State 返回全局表中记录的该连接的接收状态
func (s *UserStream) State() *store.PushedEventState {
	return userStreamStates[s.UserId]
}

// Broadcast 向所有已连接的用户传递相同的推送
func Broadcast(wrapped store.PushedEvent) {
	for _, syncUser := range userStreams {
		syncUser.Send(wrapped)
	}
}

// BroadcastAndSave 向所有已连接的用户推送信息，并保存到数据库中
func BroadcastAndSave(wrapped store.PushedEvent) error {
	err := wrapped.Insert()

	if err != nil {
		return err
	}

	Broadcast(wrapped)

	return nil
}

// Send 向该用户的 Chan 传递一个推送
func (s *UserStream) Send(wrapped store.PushedEvent) {
	s.Chan <- wrapped.SSE()
}

// SendAndSave 向该用户的 Chan 传递一个推送，并保存到数据库中
func (s *UserStream) SendAndSave(wrapped store.PushedEvent) error {
	err := wrapped.Insert()

	if err != nil {
		return err
	}

	s.Send(wrapped)

	return nil
}

var userStreams = make(map[int]*UserStream)
var userStreamStates = make(map[int]*store.PushedEventState)
var globalStreamStates = make(map[string]*store.PushedEventState)

// Create 创建一个全局流状态表项，并将顺序字段设置为 0。
func Create(taskId string) {
	var ord = 0
	globalStreamStates[taskId] = &store.PushedEventState{TaskId: &taskId, Ord: &ord}
}

// Delete 从全局流状态表中删除 taskId 对应的状态信息。
// 该函数只应该在任务结束后调用。
func Delete(taskId string) {
	delete(globalStreamStates, taskId)
}

// GetState 从全局流状态表中获取 taskId 对应的状态信息。
// 该方法不会对键的合法性做检查，调用时请确保该键未被删除。
func GetState(taskId string) *store.PushedEventState {
	return globalStreamStates[taskId]
}

// IncrOrd 为全局流状态表对应状态信息的顺序字段增加 1。
// 该方法不会对键的合法性做检查，调用时请确保该键未被删除。
func IncrOrd(taskId string) {
	globalStreamStates[taskId].IncrOrd()
}

// RegisterUser 将用户的 SSE 连接信息记录到全局表中，用于后续推送信息
func RegisterUser(userId int, conn *sse.Conn, ctx context.Context) *UserStream {
	var syncUser = &UserStream{
		UserId: userId,
		Conn:   conn,
		Chan:   make(chan *sse.Event, 16),
		Ctx:    ctx,
	}

	userStreams[userId] = syncUser
	return syncUser
}

// UnregisterUser 从全局表中删除指定用户的 SSE 连接信息
func UnregisterUser(userId int) {
	delete(userStreams, userId)
}

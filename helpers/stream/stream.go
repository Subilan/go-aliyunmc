package stream

import (
	"context"

	"github.com/Subilan/gomc-server/helpers/store"
	"go.jetify.com/sse"
)

type Stream struct {
	UserId int
	Conn   *sse.Conn
	Chan   chan *sse.Event
	Ctx    context.Context
}

func (s *Stream) State() *store.PushedEventState {
	return userStreamStates[s.UserId]
}

func BroadcastAndSave(wrapped store.PushedEvent) error {
	err := wrapped.Save()

	if err != nil {
		return err
	}

	for _, syncUser := range userStreams {
		syncUser.Send(wrapped)
	}

	return nil
}

func (s *Stream) Send(wrapped store.PushedEvent) {
	s.Chan <- wrapped.SSE()
}

func (s *Stream) SendAndSave(wrapped store.PushedEvent) error {
	err := wrapped.Save()

	if err != nil {
		return err
	}

	s.Send(wrapped)

	return nil
}

var userStreams = make(map[int]*Stream)
var userStreamStates = make(map[int]*store.PushedEventState)
var globalStreamStates = make(map[string]*store.PushedEventState)

func Create(taskId string) {
	var ord = 0
	globalStreamStates[taskId] = &store.PushedEventState{TaskId: &taskId, Ord: &ord}
}

func GetState(taskId string) *store.PushedEventState {
	return globalStreamStates[taskId]
}

func IncrOrd(taskId string) {
	globalStreamStates[taskId].IncrOrd()
}

func RegisterStream(userId int, conn *sse.Conn, ctx context.Context) *Stream {
	var syncUser = &Stream{
		UserId: userId,
		Conn:   conn,
		Chan:   make(chan *sse.Event, 16),
		Ctx:    ctx,
	}

	userStreams[userId] = syncUser
	return syncUser
}

func UnregisterStream(userId int) {
	delete(userStreams, userId)
}

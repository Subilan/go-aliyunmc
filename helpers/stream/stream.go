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

func (s *Stream) State() *State {
	return userStreamStates[s.UserId]
}

func BroadcastAndSave(wrapped Event) error {
	err := wrapped.Save()

	if err != nil {
		return err
	}

	for _, syncUser := range userStreams {
		syncUser.Send(wrapped)
	}

	return nil
}

func (s *Stream) Send(wrapped Event) {
	s.Chan <- DataEvent(wrapped)
}

func (s *Stream) SendAndSave(wrapped Event) error {
	err := wrapped.Save()

	if err != nil {
		return err
	}

	s.Send(wrapped)

	return nil
}

var userStreams = make(map[int]*Stream)
var userStreamStates = make(map[int]*State)
var globalStreamStates = make(map[string]*State)

func Create(taskId string, eventType store.PushedEventType) {
	var ord = 0
	globalStreamStates[taskId] = &State{Type: eventType, TaskId: &taskId, Ord: &ord}
}

func GetState(taskId string) *State {
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

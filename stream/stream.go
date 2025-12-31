package stream

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/Subilan/gomc-server/globals"
	"github.com/gin-gonic/gin"
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

type Event struct {
	State   *State
	IsError bool
	Content string
}

func BroadcastAndSave(wrapped Event) error {
	_, err := globals.Pool.Exec("INSERT INTO `pushed_events` (task_id, ord, `type`, is_error, content) VALUES (?, ?, ?, ?, ?)", wrapped.State.TaskId, wrapped.State.Ord, wrapped.State.Type, wrapped.IsError, wrapped.Content)

	if err != nil {
		return err
	}

	for _, syncUser := range userStreams {
		syncUser.Send(wrapped)
	}

	return nil
}

func BuildEvent(wrapped Event) *sse.Event {
	return &sse.Event{
		ID: wrapped.State.String(),
		Data: gin.H{
			"type":     wrapped.State.Type,
			"is_error": wrapped.IsError,
			"content":  wrapped.Content,
		},
	}
}

func (s *Stream) Send(wrapped Event) {
	s.Chan <- BuildEvent(wrapped)
}

func (s *Stream) SendAndSave(wrapped Event) error {
	_, err := globals.Pool.Exec("INSERT INTO `pushed_events` (task_id, ord, `type`, is_error, content) VALUES (?, ?, ?, ?, ?)", wrapped.State.TaskId, wrapped.State.Ord, wrapped.State.Type, wrapped.IsError, wrapped.Content)

	if err != nil {
		return err
	}

	s.Send(wrapped)

	return nil
}

type State struct {
	TaskId *string
	Ord    *int
	Type   Type
}

func (state *State) String() string {
	if state.TaskId == nil || state.Ord == nil {
		return ""
	}
	return *state.TaskId + "$" + strconv.Itoa(*state.Ord) + "$" + strconv.Itoa(int(state.Type))
}

func (state *State) IncrOrd() {
	if state.Ord != nil {
		*(state.Ord)++
	}
}

func StateFromString(stateStr string) (*State, error) {
	splitted := strings.Split(stateStr, "$")

	if len(splitted) != 3 {
		return nil, errors.New("invalid state string")
	}

	ord, err := strconv.Atoi(splitted[1])

	if err != nil {
		return nil, err
	}

	typ, err := strconv.Atoi(splitted[2])
	if err != nil {
		return nil, err
	}

	return &State{
		TaskId: &splitted[0],
		Ord:    &ord,
		Type:   Type(typ),
	}, nil
}

var userStreams = make(map[int]*Stream)
var userStreamStates = make(map[int]*State)
var globalStreamStates = make(map[string]*State)

type Type int

const (
	Deployment Type = iota
	Server
	Instance
)

func BeginUserStream(userId int, taskId string, streamType Type) {
	var ord = 0
	userStreamStates[userId] = &State{Type: streamType, TaskId: &taskId, Ord: &ord}
}

func GetUserStreamState(userId int) *State {
	return userStreamStates[userId]
}

func GetUserStreamOrd(userId int) *int {
	return userStreamStates[userId].Ord
}

func IncrUserStreamOrd(userId int) {
	userStreamStates[userId].IncrOrd()
}

func BeginGlobalStream(taskId string, streamType Type) {
	var ord = 0
	globalStreamStates[taskId] = &State{Type: streamType, TaskId: &taskId, Ord: &ord}
}

func GetGlobalStreamState(taskId string) *State {
	return globalStreamStates[taskId]
}

func IncrGlobalStreamOrd(taskId string) {
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

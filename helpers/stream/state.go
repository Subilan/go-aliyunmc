package stream

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Subilan/gomc-server/helpers/store"
)

type State struct {
	TaskId *string
	Ord    *int
	Type   store.PushedEventType
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
		Type:   store.PushedEventType(typ),
	}, nil
}

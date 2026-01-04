package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/gin-gonic/gin"
	"go.jetify.com/sse"
)

type PushedEvent struct {
	PushedEventState
	Type      PushedEventType `json:"type"`
	IsError   bool            `json:"isError"`
	Content   string          `json:"content"`
	CreatedAt time.Time       `json:"createdAt"`
}

type PushedEventState struct {
	TaskId *string `json:"taskId"`
	Ord    *int    `json:"ord"`
}

func (s *PushedEventState) String() string {
	if s.TaskId == nil || s.Ord == nil {
		return ""
	}

	return fmt.Sprintf("%s$%d", *s.TaskId, *s.Ord)
}

func (s *PushedEventState) IncrOrd() {
	if s.Ord != nil {
		*(s.Ord)++
	}
}

func PushedEventStateFromString(stateStr string) (*PushedEventState, error) {
	splitted := strings.Split(stateStr, "$")

	if len(splitted) != 2 {
		return nil, errors.New("invalid state string")
	}

	ord, err := strconv.Atoi(splitted[1])

	if err != nil {
		return nil, err
	}

	return &PushedEventState{
		TaskId: &splitted[0],
		Ord:    &ord,
	}, nil
}

func (event *PushedEvent) SSE() *sse.Event {
	return &sse.Event{
		ID: event.PushedEventState.String(),
		Data: gin.H{
			"type":     event.Type,
			"is_error": event.IsError,
			"content":  event.Content,
		},
	}
}

func (event *PushedEvent) Insert() error {
	_, err := globals.Pool.Exec("INSERT INTO `pushed_events` (task_id, ord, `type`, is_error, content) VALUES (?, ?, ?, ?, ?)", event.TaskId, event.Ord, event.Type, event.IsError, event.Content)

	if err != nil {
		return err
	}

	return nil
}

func GetPushedEvents(taskId string) ([]PushedEvent, error) {
	var result = make([]PushedEvent, 0)

	rows, err := globals.Pool.Query("SELECT task_id, ord, type, is_error, content FROM pushed_events WHERE task_id = ? ORDER BY ord", taskId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var event PushedEvent
		err = rows.Scan(&event.TaskId, &event.Ord, &event.Type, &event.IsError, &event.Content)

		if err != nil {
			log.Println("cannot scan row: ", err.Error())
			return nil, err
		}

		result = append(result, event)
	}

	return result, nil
}

type PushedEventType int

const (
	EventTypeDeployment PushedEventType = iota
	EventTypeServer
	EventTypeInstance
)

func BuildStatelessEvent(data any, typ PushedEventType) (PushedEvent, error) {
	marshalledData, err := json.Marshal(data)

	if err != nil {
		return PushedEvent{}, err
	}

	return PushedEvent{
		PushedEventState: PushedEventState{},
		Type:             typ,
		IsError:          false,
		Content:          string(marshalledData),
		CreatedAt:        time.Now(),
	}, nil
}

type InstanceEventType string

const (
	InstanceEventNotify                     InstanceEventType = "notify"
	InstanceEventActiveStatusUpdate         InstanceEventType = "active_status_update"
	InstanceEventActiveIpUpdate             InstanceEventType = "active_ip_update"
	InstanceEventCreated                    InstanceEventType = "created"
	InstanceEventDeploymentTaskStatusUpdate InstanceEventType = "deployment_task_status_update"
)

const (
	InstanceNotificationDeleted = "instance_deleted"
)

func BuildInstanceEvent(typ InstanceEventType, data any) (PushedEvent, error) {
	return BuildStatelessEvent(gin.H{"type": typ, "data": data}, EventTypeInstance)
}

type ServerEventType string

const (
	ServerEventNotify              ServerEventType = "notify"
	ServerEventOnlineCountUpdate   ServerEventType = "online_count_update"
	ServerEventOnlinePlayersUpdate ServerEventType = "online_players_update"
)

const (
	ServerNotificationClosed  ServerEventType = "closed"
	ServerNotificationRunning ServerEventType = "running"
)

func BuildServerEvent(typ ServerEventType, data any) (PushedEvent, error) {
	return BuildStatelessEvent(gin.H{"type": typ, "data": data}, EventTypeServer)
}

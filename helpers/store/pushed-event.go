package store

import (
	"time"
)

type PushedEvent struct {
	TaskId    string          `json:"taskId"`
	Ord       int             `json:"ord"`
	Type      PushedEventType `json:"type"`
	IsError   bool            `json:"isError"`
	Content   string          `json:"content"`
	CreatedAt time.Time       `json:"createdAt"`
}

type PushedEventType int

const (
	EventTypeDeployment PushedEventType = iota
	EventTypeServer
	EventTypeInstance
)

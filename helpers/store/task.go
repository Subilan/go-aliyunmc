package store

import "time"

type Task struct {
	TaskId    string     `json:"taskId"`
	TaskType  TaskType   `json:"taskType"`
	UserId    int        `json:"userId"`
	Status    TaskStatus `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
}

type TaskType string

const (
	TaskTypeInstanceDeployment TaskType = "instance_deployment"
)

type TaskStatus string

const (
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSuccess              = "success"
	TaskStatusFailed               = "failed"
	TaskStatusCancelled            = "cancelled"
	TaskStatusTimedOut             = "timed_out"
)

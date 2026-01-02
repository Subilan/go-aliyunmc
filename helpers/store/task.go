package store

import (
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/google/uuid"
)

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

func InsertTask(taskType TaskType, userId int) (string, error) {
	uuidS, err := uuid.NewRandom()

	if err != nil {
		return "", err
	}

	taskId := uuidS.String()

	_, err = globals.Pool.Exec("INSERT INTO tasks (task_id, `type`, user_id) VALUES (?, ?, ?)", taskId, taskType, userId)

	if err != nil {
		return "", err
	}

	return taskId, nil
}

func GetRunningTaskCount(taskType TaskType) (int, error) {
	var cnt int
	err := globals.Pool.QueryRow("SELECT COUNT(*) FROM tasks WHERE `type` = ? AND status = ?", taskType, TaskStatusRunning).Scan(&cnt)

	if err != nil {
		return 0, err
	}

	return cnt, nil
}

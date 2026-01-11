package store

import (
	"time"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/google/uuid"
)

type Task struct {
	Id        string            `json:"id"`
	Type      consts.TaskType   `json:"type"`
	UserId    int               `json:"userId"`
	Status    consts.TaskStatus `json:"status"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt *time.Time        `json:"updatedAt"`
}

type JoinedTask struct {
	Task
	Username string `json:"username"`
}

func InsertTask(taskType consts.TaskType, userId int64) (string, error) {
	uuidS, err := uuid.NewRandom()

	if err != nil {
		return "", err
	}

	taskId := uuidS.String()

	_, err = db.Pool.Exec("INSERT INTO tasks (task_id, `type`, user_id) VALUES (?, ?, ?)", taskId, taskType, userId)

	if err != nil {
		return "", err
	}

	return taskId, nil
}

func GetRunningTaskCount(taskType consts.TaskType) (int, error) {
	var cnt int
	err := db.Pool.QueryRow("SELECT COUNT(*) FROM tasks WHERE `type` = ? AND status = ?", taskType, consts.TaskStatusRunning).Scan(&cnt)

	if err != nil {
		return 0, err
	}

	return cnt, nil
}

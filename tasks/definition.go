package tasks

import (
	"go-aliyunmc/store/models"
	"time"
)

type TaskDefinition struct {
	Timeout       time.Duration
	Type          models.TaskType
	TypeExclusive bool
	F             TaskFunc
}

type TaskFunc func(*TaskContext) error

var TaskDefinitions = make(map[models.TaskType]*TaskDefinition)

func GetTaskDefinition(taskType models.TaskType) *TaskDefinition {
	def, ok := TaskDefinitions[taskType]
	if !ok {
		return nil
	}
	return def
}

func MustInitialize() {
	TaskDefinitions["test"] = &TaskDefinition{
		TypeExclusive: true,
		Type:          "test",
		Timeout:       1 * time.Hour,
		F:             testTask,
	}
}

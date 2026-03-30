package tasks

import (
	"go-aliyunmc/store/models"
	"sync"
)

var executors sync.Map
var executingTypes sync.Map

func IsTypeExecuting(taskType models.TaskType) bool {
	_, loaded := executingTypes.Load(taskType)
	return loaded
}

func RecordExecutingType(taskType models.TaskType) {
	executingTypes.Store(taskType, true)
}

func DeleteExecutingType(taskType models.TaskType) {
	executingTypes.Delete(taskType)
}

func GetExecutor(taskId uint) (*Executor, bool) {
	executor, loaded := executors.Load(taskId)
	if !loaded {
		return nil, loaded
	}
	return executor.(*Executor), true
}

func RecordExecutor(taskId uint, executor *Executor) {
	executors.Store(taskId, executor)
}

func DeleteExecutor(taskId uint) {
	executors.Delete(taskId)
}
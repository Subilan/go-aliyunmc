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

func SetExecutingType(taskType models.TaskType) {
	executingTypes.Store(taskType, true)
}

func DeleteExecutingType(taskType models.TaskType) {
	executingTypes.Delete(taskType)
}

func RangeExecutors(fn func(taskId uint, executor *Executor)) {
	executors.Range(func(key any, value any) bool {
		taskId := key.(uint)
		executor := value.(*Executor)
		fn(taskId, executor)
		return true
	})
}

func GetExecutor(taskId uint) (*Executor, bool) {
	executor, loaded := executors.Load(taskId)
	if !loaded {
		return nil, loaded
	}
	return executor.(*Executor), true
}

func SetExecutor(taskId uint, executor *Executor) {
	executors.Store(taskId, executor)
}

func DeleteExecutor(taskId uint) {
	executors.Delete(taskId)
}

package tasks

import (
	"go-aliyunmc/store/models"
	"sync"
)

var runningExecutors sync.Map
var runningExclusiveExecutors sync.Map

// SetExclusiveExecutor 将正在执行的独占任务执行器与任务类型关联起来，供后续查询和管理。
func SetExclusiveExecutor(taskType models.TaskType, executor *Executor) {
	runningExclusiveExecutors.Store(taskType, executor)
}

// GetExclusiveExecutor 获取当前正在执行的独占任务执行器，如果不存在则返回 nil 和 false。
// 只有类型互斥任务才能使用这个函数查询正在执行的任务执行器。
func GetExclusiveExecutor(taskType models.TaskType) (*Executor, bool) {
	executor, loaded := runningExclusiveExecutors.Load(taskType)
	if !loaded {
		return nil, loaded
	}
	return executor.(*Executor), true
}

// DeleteExclusiveExecutor 从正在执行的独占任务执行器表中删除指定任务类型的执行器。
func DeleteExclusiveExecutor(taskType models.TaskType) {
	runningExclusiveExecutors.Delete(taskType)
}

func IsExclusiveTaskRunning(taskType models.TaskType) bool {
	_, running := GetExclusiveExecutor(taskType)
	return running
}

func IsArchiveTaskRunning() bool {
	return IsExclusiveTaskRunning(models.TaskTypeArchive)
}

func IsBackupTaskRunning() bool {
	return IsExclusiveTaskRunning(models.TaskTypeBackup)
}

// RangeExecutors 使用 fn 遍历当前所有正在执行的任务执行器。
func RangeExecutors(fn func(taskId uint, executor *Executor)) {
	runningExecutors.Range(func(key any, value any) bool {
		taskId := key.(uint)
		executor := value.(*Executor)
		fn(taskId, executor)
		return true
	})
}

func GetExecutor(taskId uint) (*Executor, bool) {
	executor, loaded := runningExecutors.Load(taskId)
	if !loaded {
		return nil, loaded
	}
	return executor.(*Executor), true
}

func SetExecutor(taskId uint, executor *Executor) {
	runningExecutors.Store(taskId, executor)
}

func DeleteExecutor(taskId uint) {
	runningExecutors.Delete(taskId)
}

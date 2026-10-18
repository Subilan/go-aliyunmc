package tasks

import (
	"github.com/Subilan/go-aliyunmc/store/models"
	"sync"
)

var runningExecutors sync.Map
var runningExclusiveExecutors sync.Map

// GetExclusiveExecutor 获取当前持有独占 key 的执行器，如果不存在则返回 nil 和 false。
// key 通常是任务类型，也可以是多个任务共享的 ExclusiveGroup。
func GetExclusiveExecutor(key models.TaskType) (*Executor, bool) {
	executor, loaded := runningExclusiveExecutors.Load(key)
	if !loaded {
		return nil, loaded
	}
	return executor.(*Executor), true
}

// DeleteExclusiveExecutor 释放指定的独占 key。
func DeleteExclusiveExecutor(key models.TaskType) {
	runningExclusiveExecutors.Delete(key)
}

func IsExclusiveTaskRunning(key models.TaskType) bool {
	_, running := GetExclusiveExecutor(key)
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

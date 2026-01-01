package tasks

import (
	"context"
	"sync"
)

// remoteTasks 是当前正在运行的任务表。key 为任务的标识符，value 为上下文的取消函数
var remoteTasks = make(map[string]context.CancelFunc)

// mu 是用于避免同时读写 remoteTasks 的互斥锁
var mu sync.Mutex

// Register 将一个新的任务的取消函数加入到任务表中
func Register(cancel context.CancelFunc, taskId string) {
	mu.Lock()
	defer mu.Unlock()
	remoteTasks[taskId] = cancel
}

// CancelById 尝试将 taskId 指定的任务取消并从任务表中删除。如果找不到该任务，函数返回 false
func CancelById(taskId string) bool {
	mu.Lock()
	defer mu.Unlock()
	cancel, ok := remoteTasks[taskId]

	if !ok {
		return false
	}

	cancel()
	delete(remoteTasks, taskId)
	return true
}

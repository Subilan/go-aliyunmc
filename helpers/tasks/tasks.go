package tasks

import (
	"context"
	"sync"
)

var remoteTasks = make(map[string]context.CancelFunc)
var mu sync.Mutex

func Register(cancel context.CancelFunc, taskId string) {
	mu.Lock()
	defer mu.Unlock()
	remoteTasks[taskId] = cancel
}

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

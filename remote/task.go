package remote

import (
	"context"
	"sync"
)

var remoteTasks = make(map[string]context.CancelFunc)
var mu sync.Mutex

func RegisterTask(cancel context.CancelFunc, taskId string) {
	mu.Lock()
	defer mu.Unlock()
	remoteTasks[taskId] = cancel
}

func CancelTask(taskId string) bool {
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

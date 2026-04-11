package states

import "sync/atomic"

var archiveTaskRunning atomic.Bool

func IsArchiveTaskRunning() bool {
	return archiveTaskRunning.Load()
}

func SetArchiveTaskRunning(running bool) {
	archiveTaskRunning.Store(running)
}

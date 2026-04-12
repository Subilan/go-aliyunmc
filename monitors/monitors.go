package monitors

import (
	"context"
	"fmt"
	"sync"
)

var emptyValueError = fmt.Errorf("值为空")
var missingTargetError = fmt.Errorf("目标不存在")

var (
	serverMonitor   *ServerStatusMonitor
	instanceMonitor *InstanceStatusMonitor
	fileSyncPoller  *FileSyncPoller
	autoArchiveIdle *AutoArchiveIdleMonitor
	backupMonitor   *BackupMonitor
	bestEcsCandidateMonitor *BestEcsCandidateMonitor
	initializeOnce  sync.Once
)

// MustInitialize 启动所有常驻 monitor。
func MustInitialize(ctx context.Context) {
	initializeOnce.Do(func() {
		serverMonitor = newServerStatusMonitor()
		instanceMonitor = newInstanceStatusMonitor()
		fileSyncPoller = newFileSyncPoller()
		autoArchiveIdle = newAutoArchiveIdleMonitor()
		backupMonitor = newBackupMonitor()
		bestEcsCandidateMonitor = newBestEcsCandidateMonitor()

		go serverMonitor.run(ctx)
		go instanceMonitor.run(ctx)
		go fileSyncPoller.run(ctx)
		go autoArchiveIdle.run(ctx)
		go backupMonitor.run(ctx)
		go bestEcsCandidateMonitor.run(ctx)
	})
}
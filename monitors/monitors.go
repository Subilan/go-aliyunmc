package monitors

import (
	"context"
	"fmt"
	"sync"
)

var emptyValueError = fmt.Errorf("值为空")
var missingTargetError = fmt.Errorf("目标不存在")

var (
	serverMonitor           *ServerStatusMonitor
	instanceMonitor         *InstanceStatusMonitor
	fileSyncPoller          *FileSyncPoller
	autoArchiveIdle         *AutoArchiveIdleMonitor
	backupMonitor           *BackupMonitor
	bestEcsCandidateMonitor *BestEcsCandidateMonitor
	playerListSampler       *PlayerListSampler
	balanceSampler          *BalanceSampler
	initializeOnce          sync.Once
)

// GetPlayerListSampler 返回玩家列表采样器实例。
func GetPlayerListSampler() *PlayerListSampler {
	return playerListSampler
}

// GetBalanceSampler 返回余额采样器实例。
func GetBalanceSampler() *BalanceSampler {
	return balanceSampler
}

// MustInitialize 启动所有常驻 monitor 和 sampler。
func MustInitialize(ctx context.Context) {
	initializeOnce.Do(func() {
		serverMonitor = newServerStatusMonitor()
		instanceMonitor = newInstanceStatusMonitor()
		fileSyncPoller = newFileSyncPoller()
		autoArchiveIdle = newAutoArchiveIdleMonitor()
		backupMonitor = newBackupMonitor()
		bestEcsCandidateMonitor = newBestEcsCandidateMonitor()
		playerListSampler = newPlayerListSampler()
		balanceSampler = newBalanceSampler()

		go serverMonitor.run(ctx)
		go instanceMonitor.run(ctx)
		go fileSyncPoller.run(ctx)
		go autoArchiveIdle.run(ctx)
		go backupMonitor.run(ctx)
		go bestEcsCandidateMonitor.run(ctx)
		go playerListSampler.run(ctx)
		go balanceSampler.run(ctx)
	})
}

package monitors

import (
	"context"
	"fmt"
	"github.com/Subilan/go-aliyunmc/tasks"
	"sync"
)

var emptyValueError = fmt.Errorf("empty_value")
var missingTargetError = fmt.Errorf("missing_target")

var (
	serverMonitor           *ServerStatusMonitor
	instanceMonitor         *InstanceStatusMonitor
	fileSyncPoller          *FileSyncPoller
	autoArchiveIdle         *AutoArchiveIdleMonitor
	backupMonitor           *BackupMonitor
	spotInterruptionMonitor *SpotInterruptionMonitor
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

// GetServerStatusMonitor 返回服务器状态监控器实例。
func GetServerStatusMonitor() *ServerStatusMonitor {
	return serverMonitor
}

// GetInstanceStatusMonitor 返回实例状态监控器实例。
func GetInstanceStatusMonitor() *InstanceStatusMonitor {
	return instanceMonitor
}

// GetBestEcsCandidateMonitor 返回最佳ECS候选监控器实例。
func GetBestEcsCandidateMonitor() *BestEcsCandidateMonitor {
	return bestEcsCandidateMonitor
}

// GetSpotInterruptionMonitor 返回抢占式实例回收监控器实例。
func GetSpotInterruptionMonitor() *SpotInterruptionMonitor {
	return spotInterruptionMonitor
}

// MustInitialize 启动所有常驻 monitor 和 sampler。
func MustInitialize(ctx context.Context) {
	initializeOnce.Do(func() {
		serverMonitor = newServerStatusMonitor()
		instanceMonitor = newInstanceStatusMonitor()
		fileSyncPoller = newFileSyncPoller()
		autoArchiveIdle = newAutoArchiveIdleMonitor()
		backupMonitor = newBackupMonitor()
		spotInterruptionMonitor = newSpotInterruptionMonitor()
		bestEcsCandidateMonitor = newBestEcsCandidateMonitor()
		playerListSampler = newPlayerListSampler()
		balanceSampler = newBalanceSampler()

		tasks.SnapshotServerStatus = serverMonitor.Snapshot
		tasks.SnapshotBestEcsCandidate = bestEcsCandidateMonitor.Snapshot
		tasks.WaitInstanceSnapshot = instanceMonitor.WaitSnapshot

		go serverMonitor.run(ctx)
		go instanceMonitor.run(ctx)
		go fileSyncPoller.run(ctx)
		go autoArchiveIdle.run(ctx)
		go backupMonitor.run(ctx)
		go spotInterruptionMonitor.run(ctx)
		go bestEcsCandidateMonitor.run(ctx)
		go playerListSampler.run(ctx)
		go balanceSampler.run(ctx)
	})
}

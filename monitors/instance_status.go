package monitors

import (
	"context"
	"fmt"
	"time"

	"go-aliyunmc/aliyun"
	"go-aliyunmc/env"
	"go-aliyunmc/logs"
	"go-aliyunmc/store"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

type InstanceStatusMonitor struct {
	interval time.Duration
	st       *snapshotStore[string]
	hub      *hub[snapshot[string]]
	logger   *logs.PrefixedLogger
}

func newInstanceStatusMonitor() *InstanceStatusMonitor {
	return &InstanceStatusMonitor{
		interval: time.Duration(InstanceStatusC.PollIntervalSec) * time.Second,
		hub:      newHub[snapshot[string]](),
		st:       &snapshotStore[string]{},
		logger:   logs.NewPrefixedLogger("[monitor/instance] "),
	}
}

func (m *InstanceStatusMonitor) Snapshot() snapshot[string] {
	return m.st.Snapshot()
}

func (m *InstanceStatusMonitor) Subscribe() (<-chan snapshot[string], func()) {
	return m.hub.Subscribe()
}

func (m *InstanceStatusMonitor) run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	m.pollAndStore(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pollAndStore(ctx)
		}
	}
}

func (m *InstanceStatusMonitor) pollAndStore(ctx context.Context) {
	instance, err := store.GetActiveInstanceDefaultNil()
	if err != nil {
		m.st.StoreError(err, m.hub, m.logger)
		return
	}

	if instance == nil {
		m.st.StoreError(missingTargetError, m.hub, m.logger)
		return
	}

	if aliyun.EcsClient == nil {
		m.st.StoreError(fmt.Errorf("EcsClient is nil"), m.hub, m.logger)
		return
	}

	resp, err := aliyun.EcsClient.DescribeInstanceStatusWithContext(ctx, &ecs20140526.DescribeInstanceStatusRequest{
		RegionId:   tea.String(instance.RegionId),
		InstanceId: []*string{tea.String(instance.InstanceId)},
	}, &dara.RuntimeOptions{})

	if err != nil {
		if env.DEV {
			m.logger.Warn("DescribeInstanceStatus失败: %v", err)
		}
		m.st.StoreError(err, m.hub, m.logger)
		return
	}

	if resp == nil || resp.Body == nil || resp.Body.InstanceStatuses == nil {
		if env.DEV {
			m.logger.Warn("ECS返回空状态")
		}
		m.st.StoreError(emptyValueError, m.hub, m.logger)
		return
	}

	instanceStatuses := resp.Body.InstanceStatuses.InstanceStatus
	for _, statusItem := range instanceStatuses {
		if statusItem != nil && statusItem.GetInstanceId() != nil && *statusItem.GetInstanceId() == instance.InstanceId && statusItem.GetStatus() != nil {
			m.st.Store(*statusItem.GetStatus(), m.hub, m.logger)
			return
		}
	}

	m.st.StoreError(emptyValueError, m.hub, m.logger)
}

// SnapshotInstanceStatus 返回当前 instance status 副本。
func SnapshotInstanceStatus() snapshot[string] {
	if instanceMonitor == nil {
		return snapshot[string]{}
	}
	return instanceMonitor.Snapshot()
}

// SnapshotIsInstanceRunning 快速检查实例是否处于 Running 状态。该函数不会等待 instance status monitor 首次获取结果，因此请避免在系统启动时调用。
func SnapshotIsInstanceRunning() bool {
	snapshot := SnapshotInstanceStatus()
	return snapshot.Error == nil && snapshot.Value == "Running"
}

// StableSnapshotIsInstanceRunning 在 instance monitor 尚未产出首个快照时等待至多 timeout。
// 一旦有快照（无论值是 Running 还是其他状态），行为与 SnapshotIsInstanceRunning 保持一致。
func StableSnapshotIsInstanceRunning(timeout time.Duration) bool {
	if instanceMonitor == nil {
		return SnapshotIsInstanceRunning()
	}

	current := SnapshotInstanceStatus()
	if !current.UpdatedAt.IsZero() {
		return current.Error == nil && current.Value == "Running"
	}

	if timeout <= 0 {
		return SnapshotIsInstanceRunning()
	}

	ch, unsubscribe := SubscribeInstanceSnapshot()
	defer unsubscribe()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		current = SnapshotInstanceStatus()
		if !current.UpdatedAt.IsZero() {
			return current.Error == nil && current.Value == "Running"
		}

		select {
		case <-timer.C:
			return SnapshotIsInstanceRunning()
		case <-ch:
		}
	}
}

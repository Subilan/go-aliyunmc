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
}

func newInstanceStatusMonitor(cfg InstanceStatusMonitorConfig) *InstanceStatusMonitor {
	return &InstanceStatusMonitor{
		interval: time.Duration(cfg.PollIntervalSec) * time.Second,
		hub:      newHub[snapshot[string]](),
		st:       &snapshotStore[string]{},
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
		m.st.StoreError(err, m.hub)
		return
	}

	if instance == nil {
		m.st.StoreError(missingTargetError, m.hub)
		return
	}

	if aliyun.EcsClient == nil {
		m.st.StoreError(fmt.Errorf("EcsClient is nil"), m.hub)
		return
	}

	resp, err := aliyun.EcsClient.DescribeInstanceStatusWithContext(ctx, &ecs20140526.DescribeInstanceStatusRequest{
		RegionId:   tea.String(instance.RegionId),
		InstanceId: []*string{tea.String(instance.InstanceId)},
	}, &dara.RuntimeOptions{})

	if err != nil {
		if env.DEV {
			logs.Warn("[monitor/instance] DescribeInstanceStatus失败: %v", err)
		}
		m.st.StoreError(err, m.hub)
		return
	}

	if resp == nil || resp.Body == nil || resp.Body.InstanceStatuses == nil {
		if env.DEV {
			logs.Warn("[monitor/instance] ECS返回空状态")
		}
		m.st.StoreError(emptyValueError, m.hub)
		return
	}

	instanceStatuses := resp.Body.InstanceStatuses.InstanceStatus
	for _, statusItem := range instanceStatuses {
		if statusItem != nil && statusItem.GetInstanceId() != nil && *statusItem.GetInstanceId() == instance.InstanceId && statusItem.GetStatus() != nil {
			m.st.Store(*statusItem.GetStatus(), m.hub)
			return
		}
	}

	m.st.StoreError(emptyValueError, m.hub)
}

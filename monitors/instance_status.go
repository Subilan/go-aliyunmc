package monitors

import (
	"context"
	"fmt"
	"time"

	"go-aliyunmc/aliyun"
	"go-aliyunmc/env"
	"go-aliyunmc/log_util"
	"go-aliyunmc/states"
	"go-aliyunmc/store"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

type InstanceStatusMonitor struct {
	store    *states.HubbedStore[string]
	interval time.Duration
	logger   *log_util.NamedLogger
}

func newInstanceStatusMonitor() *InstanceStatusMonitor {
	monitor := &InstanceStatusMonitor{
		interval: time.Duration(InstanceStatusC.PollIntervalSec) * time.Second,
		store:    states.NewHubbedStore[string](),
		logger:   log_util.NewNamedLogger("[monitor/instance] ", "instance-status-monitor"),
	}
	return monitor
}

func (m *InstanceStatusMonitor) Snapshot() states.State[string] {
	return m.store.Snapshot()
}

func (m *InstanceStatusMonitor) Subscribe() (<-chan states.State[string], func()) {
	return m.store.Subscribe()
}

func (m *InstanceStatusMonitor) WaitSnapshot(timeout time.Duration) (states.State[string], error) {
	return m.store.WaitSnapshot(timeout)
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
		m.store.StoreError(err, m.logger)
		return
	}

	if instance == nil {
		m.store.StoreError(missingTargetError, m.logger)
		return
	}

	if aliyun.EcsClient == nil {
		m.store.StoreError(fmt.Errorf("EcsClient is nil"), m.logger)
		return
	}

	resp, err := aliyun.EcsClient.DescribeInstanceStatusWithContext(ctx, &ecs20140526.DescribeInstanceStatusRequest{
		RegionId:   tea.String(aliyun.C.RegionId),
		InstanceId: []*string{tea.String(instance.InstanceId)},
	}, &dara.RuntimeOptions{})

	if err != nil {
		if env.DEV {
			m.logger.Warn("DescribeInstanceStatus失败: %v", err)
		}
		m.store.StoreError(err, m.logger)
		return
	}

	if resp == nil || resp.Body == nil || resp.Body.InstanceStatuses == nil {
		if env.DEV {
			m.logger.Warn("ECS返回空状态")
		}
		m.store.StoreError(emptyValueError, m.logger)
		return
	}

	if len(resp.Body.InstanceStatuses.InstanceStatus) == 0 {
		m.logger.Warn("未查询到实例，这表示实例已经被外部删除")
		m.logger.Warn("将删除数据库实例")
		err := store.DeleteActiveInstance()
		if err != nil {
			m.logger.Error("删除数据库实例失败: %v", err)
		}
		m.store.StoreError(emptyValueError, m.logger)
		return
	}

	instanceStatuses := resp.Body.InstanceStatuses.InstanceStatus
	for _, statusItem := range instanceStatuses {
		if statusItem != nil && statusItem.GetInstanceId() != nil && *statusItem.GetInstanceId() == instance.InstanceId && statusItem.GetStatus() != nil {
			m.store.Store(*statusItem.GetStatus(), m.logger)
			return
		}
	}

	m.store.StoreError(emptyValueError, m.logger)
}

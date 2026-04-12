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
	logger   *log_util.PrefixedLogger
}

func newInstanceStatusMonitor() *InstanceStatusMonitor {
	monitor := &InstanceStatusMonitor{
		interval: time.Duration(InstanceStatusC.PollIntervalSec) * time.Second,
		store:    states.NewRecordedHubbedStore[string](states.HSKeyInstanceStatus),
		logger:   log_util.NewPrefixedLogger("[monitor/instance] "),
	}
	return monitor
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
		RegionId: tea.String(aliyun.C.RegionId),
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

	instanceStatuses := resp.Body.InstanceStatuses.InstanceStatus
	for _, statusItem := range instanceStatuses {
		if statusItem != nil && statusItem.GetInstanceId() != nil && *statusItem.GetInstanceId() == instance.InstanceId && statusItem.GetStatus() != nil {
			m.store.Store(*statusItem.GetStatus(), m.logger)
			return
		}
	}

	m.store.StoreError(emptyValueError, m.logger)
}
package monitors

import (
	"context"
	"time"

	"go-aliyunmc/log_util"
	"go-aliyunmc/server"
	"go-aliyunmc/states"
	"go-aliyunmc/store"

	"github.com/mcstatus-io/mcutil/v4/status"
)

type ServerStatusMonitor struct {
	interval time.Duration
	store    *states.HubbedStore[states.ServerStatusState]
	logger   *log_util.NamedLogger
}

func newServerStatusMonitor() *ServerStatusMonitor {
	return &ServerStatusMonitor{
		interval: time.Duration(ServerStatusC.PollIntervalSec) * time.Second,
		store:    states.NewHubbedStore[states.ServerStatusState](),
		logger:   log_util.NewNamedLogger("[monitor/server] ", "server-status-monitor"),
	}
}

func (m *ServerStatusMonitor) Snapshot() states.State[states.ServerStatusState] {
	return m.store.Snapshot()
}

func (m *ServerStatusMonitor) Subscribe() (<-chan states.State[states.ServerStatusState], func()) {
	return m.store.Subscribe()
}

func (m *ServerStatusMonitor) run(ctx context.Context) {
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

func (m *ServerStatusMonitor) pollAndStore(ctx context.Context) {
	ip, err := store.GetActiveInstanceIpNonEmpty()

	if err != nil {
		m.store.StoreError(missingTargetError, m.logger)
		return
	}

	statusCtx, statusCtxCancel := context.WithTimeout(ctx, 5*time.Second)
	statusValue, err := status.Modern(statusCtx, ip, server.C.PortOrDefault())
	statusCtxCancel()

	next := states.ServerStatusState{}

	// 认为离线，不记录错误
	if err != nil {
		next.Online = false
		next.PlayerCount = 0
		m.store.ForceStore(next, m.logger)
		return
	}

	if statusValue != nil {
		next.Online = true
		if statusValue.Players.Online != nil {
			next.PlayerCount = *statusValue.Players.Online
		}
	}

	m.store.Store(next, m.logger)
}

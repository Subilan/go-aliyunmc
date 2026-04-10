package monitors

import (
	"context"
	"time"

	"go-aliyunmc/logs"
	"go-aliyunmc/server"
	"go-aliyunmc/store"

	"github.com/mcstatus-io/mcutil/v4/status"
)

type ServerStatusState struct {
	Online      bool  `json:"online"`
	PlayerCount int64 `json:"playerCount"`
}

type ServerStatusMonitor struct {
	interval time.Duration
	st       *snapshotStore[ServerStatusState]
	hub      *hub[snapshot[ServerStatusState]]
	logger   *logs.PrefixedLogger
}

func newServerStatusMonitor() *ServerStatusMonitor {
	return &ServerStatusMonitor{
		interval: time.Duration(ServerStatusC.PollIntervalSec) * time.Second,
		hub:      newHub[snapshot[ServerStatusState]](),
		st:       &snapshotStore[ServerStatusState]{},
		logger:   logs.NewPrefixedLogger("[monitor/server] "),
	}
}

func (m *ServerStatusMonitor) Snapshot() snapshot[ServerStatusState] {
	return m.st.Snapshot()
}

func (m *ServerStatusMonitor) Subscribe() (<-chan snapshot[ServerStatusState], func()) {
	return m.hub.Subscribe()
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
		m.st.StoreError(missingTargetError, m.hub, m.logger)
		return
	}

	statusCtx, statusCtxCancel := context.WithTimeout(ctx, 5*time.Second)
	statusValue, err := status.Modern(statusCtx, ip, server.C.PortOrDefault())
	statusCtxCancel()

	next := ServerStatusState{}

	if err != nil {
		next.Online = false
		next.PlayerCount = 0
		m.st.Store(next, m.hub, m.logger)
		return
	}

	if statusValue != nil {
		next.Online = true
		if statusValue.Players.Online != nil {
			next.PlayerCount = *statusValue.Players.Online
		}
	}

	m.st.Store(next, m.hub, m.logger)
}

// SnapshotIsServerOnline 返回当前服务器是否在线。只有在成功获取服务器状态且服务器在线时才返回 true。
func SnapshotIsServerOnline() bool {
	snapshot := SnapshotServerStatus()
	return snapshot.Error == nil && snapshot.Value.Online
}

// SnapshotIsServerOffline 返回当前服务器是否离线。只有在成功获取服务器状态且服务器离线时才返回 true。
func SnapshotIsServerOffline() bool {
	snapshot := SnapshotServerStatus()
	return snapshot.Error == nil && !snapshot.Value.Online
}

package monitors

import (
	"context"
	"time"

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
}

func newServerStatusMonitor() *ServerStatusMonitor {
	return &ServerStatusMonitor{
		interval: time.Duration(ServerStatusC.PollIntervalSec) * time.Second,
		hub:      newHub[snapshot[ServerStatusState]](),
		st:       &snapshotStore[ServerStatusState]{},
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
		m.st.StoreError(missingTargetError, m.hub)
		return
	}

	statusCtx, statusCtxCancel := context.WithTimeout(ctx, 5*time.Second)
	statusValue, err := status.Modern(statusCtx, ip, server.C.PortOrDefault())
	statusCtxCancel()

	next := ServerStatusState{}

	if err != nil {
		next.Online = false
		next.PlayerCount = 0
		m.st.Store(next, m.hub)
		return
	}

	if statusValue != nil {
		next.Online = true
		if statusValue.Players.Online != nil {
			next.PlayerCount = *statusValue.Players.Online
		}
	}

	m.st.Store(next, m.hub)
}

package monitors

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/messages"
)

const (
	// playerPresenceCollectInterval 是读取最新采样点的间隔。
	playerPresenceCollectInterval = time.Second
	// playerPresenceMinGap 是采样断档判定的下限，避免采样间隔过短时误判。
	playerPresenceMinGap = 15 * time.Second
)

// playerPresenceMessageData 是 player_joined / player_left 消息模板可用的变量。
type playerPresenceMessageData struct {
	// Players 是以「、」分隔的玩家名列表。
	Players string
}

// PlayerPresenceMonitor 复用玩家列表采样器的数据，合并播报玩家进出服务器。
// 它不额外发起查询，只读取 [PlayerListSampler] 已经产生的采样点。
type PlayerPresenceMonitor struct {
	enabled       bool
	sampler       *PlayerListSampler
	serverStatus  *ServerStatusMonitor
	batchInterval time.Duration
	tracker       playerPresenceTracker
	logger        *log_util.NamedLogger
}

func newPlayerPresenceMonitor(sampler *PlayerListSampler, serverStatus *ServerStatusMonitor) *PlayerPresenceMonitor {
	return &PlayerPresenceMonitor{
		enabled:       config.Get().Monitors.PlayerPresence.Enabled,
		sampler:       sampler,
		serverStatus:  serverStatus,
		batchInterval: time.Duration(config.Get().Monitors.PlayerPresence.BatchSec) * time.Second,
		tracker: playerPresenceTracker{
			gapThreshold: playerPresenceGapThreshold(sampler.Interval()),
		},
		logger: log_util.NewNamedLogger("[monitor/player-presence] ", "player-presence-monitor"),
	}
}

// playerPresenceGapThreshold 返回采样断档阈值。超过该间隔说明中间可能漏掉了
// 有效的玩家变化（例如服务器重启或查询长时间失败），此时只重建基线。
func playerPresenceGapThreshold(sampleInterval time.Duration) time.Duration {
	gap := 3 * sampleInterval
	if gap < playerPresenceMinGap {
		return playerPresenceMinGap
	}
	return gap
}

func (m *PlayerPresenceMonitor) run(ctx context.Context) {
	if !m.enabled {
		m.logger.Info("disabled")
		return
	}

	collectTicker := time.NewTicker(playerPresenceCollectInterval)
	defer collectTicker.Stop()
	flushTicker := time.NewTicker(m.batchInterval)
	defer flushTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-collectTicker.C:
			m.collectLatest()
		case <-flushTicker.C:
			m.flush()
		}
	}
}

// collectLatest 把最新采样点并入当前集合。服务器离线时重建基线，
// 避免停服造成的全员“离开”被播报。
func (m *PlayerPresenceMonitor) collectLatest() {
	if !m.onlineOrRebaseline() {
		return
	}

	point, ok := m.sampler.Latest()
	if !ok || !m.tracker.HasNewSample(point.Time) {
		return
	}

	m.tracker.Observe(point.Time, playerSet(point.Players))
}

// flush 播报自上次播报以来的净变化；同一窗口内的多个玩家会合并成一条消息。
func (m *PlayerPresenceMonitor) flush() {
	if !m.onlineOrRebaseline() {
		return
	}

	joined, left := m.tracker.Flush()
	m.announce(joined, messages.PlayerJoined, "加入")
	m.announce(left, messages.PlayerLeft, "离开")
}

func (m *PlayerPresenceMonitor) announce(names []string, templateKey, verb string) {
	if len(names) == 0 {
		return
	}

	players := strings.Join(names, "、")
	m.logger.Info("播报玩家%s: %s", verb, players)
	messages.Send(templateKey, playerPresenceMessageData{Players: players})
}

// onlineOrRebaseline 只在服务器在线时返回 true；离线时重建基线，
// 避免停服造成的全员“离开”被播报。
func (m *PlayerPresenceMonitor) onlineOrRebaseline() bool {
	snap := m.serverStatus.Snapshot()
	if snap.IsValid() && snap.Value.Online {
		return true
	}
	m.tracker.Rebaseline()
	return false
}

// playerPresenceTracker 维护播报基线（上次播报时的在线集合）与当前集合。
type playerPresenceTracker struct {
	initialized  bool
	baseline     map[string]struct{}
	current      map[string]struct{}
	lastSampleAt time.Time
	gapThreshold time.Duration
}

// HasNewSample 报告采样点是否比上一次处理的新，避免重复解析同一份名单。
func (t *playerPresenceTracker) HasNewSample(at time.Time) bool {
	return at.After(t.lastSampleAt)
}

// Observe 处理一个采样点。首次采样、重建基线后的首个采样或采样断档时，
// 只更新基线，不产生播报内容。
func (t *playerPresenceTracker) Observe(at time.Time, players map[string]struct{}) {
	if !t.initialized || at.Sub(t.lastSampleAt) > t.gapThreshold {
		t.baseline = players
		t.current = players
		t.lastSampleAt = at
		t.initialized = true
		return
	}

	if !at.After(t.lastSampleAt) {
		return
	}

	t.current = players
	t.lastSampleAt = at
}

// Rebaseline 清空状态，让下一个采样点重新建立基线。
// 服务器离线或停服时调用，避免把停机当成大量玩家离开。
func (t *playerPresenceTracker) Rebaseline() {
	t.initialized = false
	t.baseline = nil
	t.current = nil
	t.lastSampleAt = time.Time{}
}

// Flush 返回自上次播报以来的净变化，并把当前集合推进为新的基线。
func (t *playerPresenceTracker) Flush() (joined, left []string) {
	if !t.initialized {
		return nil, nil
	}

	joined = playerNameDifference(t.current, t.baseline)
	left = playerNameDifference(t.baseline, t.current)
	t.baseline = t.current
	return joined, left
}

// playerSet 把采样器解析后的玩家名列表转换成集合。
func playerSet(names []string) map[string]struct{} {
	players := make(map[string]struct{}, len(names))
	for _, name := range names {
		players[name] = struct{}{}
	}
	return players
}

// playerNameDifference 返回在 a 中但不在 b 中的玩家名，按字典序排序。
func playerNameDifference(a, b map[string]struct{}) []string {
	var names []string
	for name := range a {
		if _, ok := b[name]; !ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

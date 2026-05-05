package monitors

import (
	"context"
	"strings"
	"sync"
	"time"

	"go-aliyunmc/log_util"
	"go-aliyunmc/server"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"

	"github.com/mcstatus-io/mcutil/v4/query"
)

// PlayerListDataPoint 表示一个玩家列表采样数据点。
type PlayerListDataPoint struct {
	Time        time.Time `json:"time"`
	PlayerNames string    `json:"playerNames"`
}

// PlayerListSampler 定期采样在线玩家列表。
type PlayerListSampler struct {
	interval      time.Duration
	maxDataPoints int
	dataPoints    []PlayerListDataPoint
	mu            sync.RWMutex
	logger        *log_util.NamedLogger
}

func newPlayerListSampler() *PlayerListSampler {
	s := &PlayerListSampler{
		interval:      time.Duration(PlayerCountSamplerC.SampleIntervalSec) * time.Second,
		maxDataPoints: PlayerCountSamplerC.MaxDataPoints,
		dataPoints:    make([]PlayerListDataPoint, 0, PlayerCountSamplerC.MaxDataPoints),
		logger:        log_util.NewNamedLogger("[sampler/player-list] ", "player-list-sampler"),
	}
	s.loadFromDB()
	return s
}

func (s *PlayerListSampler) loadFromDB() {
	var records []models.PlayerListSample
	if err := store.DB.Order("created_at DESC").Limit(s.maxDataPoints).Find(&records).Error; err != nil {
		s.logger.Error("加载历史玩家列表数据失败: %v", err)
		return
	}
	for i := len(records) - 1; i >= 0; i-- {
		s.dataPoints = append(s.dataPoints, PlayerListDataPoint{
			Time:        records[i].CreatedAt,
			PlayerNames: records[i].PlayerNames,
		})
	}
}

// Snapshot 返回当前内存中所有数据点的副本。
func (s *PlayerListSampler) Snapshot() []PlayerListDataPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]PlayerListDataPoint, len(s.dataPoints))
	copy(result, s.dataPoints)
	return result
}

func (s *PlayerListSampler) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.sampleAndStore(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sampleAndStore(ctx)
		}
	}
}

func (s *PlayerListSampler) sampleAndStore(ctx context.Context) {
	ip, err := store.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return
	}

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := query.Full(queryCtx, ip, server.C.PortOrDefault())
	if err != nil {
		return
	}

	playerNames := strings.Join(result.Players, ",")
	now := time.Now()
	point := PlayerListDataPoint{
		Time:        now,
		PlayerNames: playerNames,
	}

	// 持久化到数据库
	record := models.PlayerListSample{
		CreatedAt:   now,
		PlayerNames: playerNames,
	}
	if err := store.DB.Create(&record).Error; err != nil {
		s.logger.Error("持久化玩家列表采样数据失败: %v", err)
	}

	// 存入内存
	s.mu.Lock()
	s.dataPoints = append(s.dataPoints, point)
	if len(s.dataPoints) > s.maxDataPoints {
		s.dataPoints = s.dataPoints[len(s.dataPoints)-s.maxDataPoints:]
	}
	s.mu.Unlock()
}

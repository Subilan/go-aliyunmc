package monitors

import (
	"context"
	"strconv"
	"sync"
	"time"

	"go-aliyunmc/aliyun"
	"go-aliyunmc/log_util"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"
)

// BalanceDataPoint 表示一个余额采样数据点。
type BalanceDataPoint struct {
	Time   time.Time `json:"time"`
	Amount float64   `json:"amount"`
}

// BalanceSampler 定期采样阿里云账户余额。
type BalanceSampler struct {
	interval      time.Duration
	maxDataPoints int
	dataPoints    []BalanceDataPoint
	mu            sync.RWMutex
	logger        *log_util.NamedLogger
}

func newBalanceSampler() *BalanceSampler {
	s := &BalanceSampler{
		interval:      time.Duration(BalanceSamplerC.SampleIntervalSec) * time.Second,
		maxDataPoints: BalanceSamplerC.MaxDataPoints,
		dataPoints:    make([]BalanceDataPoint, 0, BalanceSamplerC.MaxDataPoints),
		logger:        log_util.NewNamedLogger("[sampler/balance] ", "balance-sampler"),
	}
	s.loadFromDB()
	return s
}

func (s *BalanceSampler) loadFromDB() {
	var records []models.BalanceSample
	if err := store.DB.Order("created_at DESC").Limit(s.maxDataPoints).Find(&records).Error; err != nil {
		s.logger.Error("加载历史余额数据失败: %v", err)
		return
	}
	for i := len(records) - 1; i >= 0; i-- {
		s.dataPoints = append(s.dataPoints, BalanceDataPoint{
			Time:   records[i].CreatedAt,
			Amount: records[i].Amount,
		})
	}
}

// Snapshot 返回当前内存中所有数据点的副本。
func (s *BalanceSampler) Snapshot() []BalanceDataPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]BalanceDataPoint, len(s.dataPoints))
	copy(result, s.dataPoints)
	return result
}

func (s *BalanceSampler) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.sampleAndStore()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sampleAndStore()
		}
	}
}

func (s *BalanceSampler) sampleAndStore() {
	response, err := aliyun.BssClient.QueryAccountBalance()
	if err != nil {
		s.logger.Error("查询账户余额失败: %v", err)
		return
	}

	amount, err := strconv.ParseFloat(*response.Body.Data.AvailableAmount, 64)
	if err != nil {
		s.logger.Error("解析余额金额失败: %v", err)
		return
	}

	now := time.Now()
	point := BalanceDataPoint{
		Time:   now,
		Amount: amount,
	}

	// 持久化到数据库
	record := models.BalanceSample{
		CreatedAt: now,
		Amount:    amount,
	}
	if err := store.DB.Create(&record).Error; err != nil {
		s.logger.Error("持久化余额采样数据失败: %v", err)
	}

	// 存入内存
	s.mu.Lock()
	s.dataPoints = append(s.dataPoints, point)
	if len(s.dataPoints) > s.maxDataPoints {
		s.dataPoints = s.dataPoints[len(s.dataPoints)-s.maxDataPoints:]
	}
	s.mu.Unlock()
}

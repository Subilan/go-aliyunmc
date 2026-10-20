package monitors

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/aliyun"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/log_util"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
)

// BalanceDataPoint 表示一个余额采样数据点。
type BalanceDataPoint struct {
	Time   time.Time `json:"time"`
	Amount float64   `json:"amount"`
}

// BalanceStatsMinSpan 是计算平均消耗速率所需的最短累计运行时长。
// 按小时计费时，窗口过短会让单个计费事件主导结果。
const BalanceStatsMinSpan = 6 * time.Hour

// BalanceStats 描述采样窗口内的余额消耗统计。
type BalanceStats struct {
	// Rate 为实例运行期间的平均消耗速率（¥/h）；运行样本不足或无净消耗时为 nil。
	Rate *float64 `json:"rate"`
	// SpanHours 是首尾采样点的总跨度。
	SpanHours float64 `json:"span_hours"`
	// RunningHours 是参与统计的实例运行时长，即与实例存在时间重叠的采样区间之和。
	RunningHours     float64 `json:"running_hours"`
	SampleCount      int     `json:"sample_count"`
	HasEnoughSamples bool    `json:"has_enough_samples"`
	MinRunningHours  float64 `json:"min_running_hours"`
}

type timeRange struct {
	start time.Time
	end   time.Time
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
		interval:      time.Duration(config.Get().Samplers.Balance.SampleIntervalSec) * time.Second,
		maxDataPoints: config.Get().Samplers.Balance.MaxDataPoints,
		dataPoints:    make([]BalanceDataPoint, 0, config.Get().Samplers.Balance.MaxDataPoints),
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

// Stats 返回当前采样窗口内、实例运行期间的余额消耗统计。
func (s *BalanceSampler) Stats() BalanceStats {
	points := s.Snapshot()
	if len(points) < 2 {
		return calculateBalanceStats(points, nil)
	}
	return calculateBalanceStats(points, s.instanceRanges(points[0].Time, points[len(points)-1].Time))
}

// instanceRanges 返回与采样窗口有重叠的实例存在区间。
func (s *BalanceSampler) instanceRanges(from, to time.Time) []timeRange {
	var instances []models.Instance
	err := store.DB.Unscoped().
		Where("created_at <= ? AND (deleted_at IS NULL OR deleted_at >= ?)", to, from).
		Find(&instances).Error
	if err != nil {
		s.logger.Error("查询实例运行时段失败: %v", err)
		return nil
	}

	now := time.Now()
	ranges := make([]timeRange, 0, len(instances))
	for _, instance := range instances {
		end := now
		if instance.DeletedAt.Valid {
			end = instance.DeletedAt.Time
		}
		ranges = append(ranges, timeRange{start: instance.CreatedAt, end: end})
	}
	return ranges
}

// calculateBalanceStats 只统计与实例运行时段重叠的采样区间，并以这些区间的
// 总时长为分母累计所有下降额。若不按实例存在时间过滤，停机期间的极低消耗
// 会把速率稀释到与实际运行成本无关的水平。
func calculateBalanceStats(points []BalanceDataPoint, running []timeRange) BalanceStats {
	stats := BalanceStats{
		SampleCount:     len(points),
		MinRunningHours: BalanceStatsMinSpan.Hours(),
	}
	if len(points) < 2 {
		return stats
	}

	sorted := append([]BalanceDataPoint(nil), points...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Time.Before(sorted[j].Time)
	})
	stats.SpanHours = sorted[len(sorted)-1].Time.Sub(sorted[0].Time).Hours()

	ranges := mergeTimeRanges(running)

	var totalDrop, runningHours float64
	for i := 1; i < len(sorted); i++ {
		start, end := sorted[i-1].Time, sorted[i].Time
		if !end.After(start) || !overlapsAny(start, end, ranges) {
			continue
		}
		runningHours += end.Sub(start).Hours()
		// 只累计下降额，忽略充值造成的上升，避免充值抵消当期消耗。
		if drop := sorted[i-1].Amount - sorted[i].Amount; drop > 0 {
			totalDrop += drop
		}
	}

	stats.RunningHours = runningHours
	if runningHours < BalanceStatsMinSpan.Hours() {
		return stats
	}
	stats.HasEnoughSamples = true
	if totalDrop > 0 {
		rate := totalDrop / runningHours
		stats.Rate = &rate
	}

	return stats
}

func mergeTimeRanges(ranges []timeRange) []timeRange {
	valid := make([]timeRange, 0, len(ranges))
	for _, r := range ranges {
		if r.end.After(r.start) {
			valid = append(valid, r)
		}
	}
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].start.Before(valid[j].start)
	})

	merged := make([]timeRange, 0, len(valid))
	for _, r := range valid {
		if len(merged) == 0 || r.start.After(merged[len(merged)-1].end) {
			merged = append(merged, r)
			continue
		}
		if r.end.After(merged[len(merged)-1].end) {
			merged[len(merged)-1].end = r.end
		}
	}
	return merged
}

func overlapsAny(start, end time.Time, ranges []timeRange) bool {
	for _, r := range ranges {
		if start.Before(r.end) && end.After(r.start) {
			return true
		}
	}
	return false
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

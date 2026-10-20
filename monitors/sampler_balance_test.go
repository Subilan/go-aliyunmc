package monitors

import (
	"math"
	"testing"
	"time"
)

func balancePoints(base time.Time, step time.Duration, amounts ...float64) []BalanceDataPoint {
	points := make([]BalanceDataPoint, len(amounts))
	for i, amount := range amounts {
		points[i] = BalanceDataPoint{
			Time:   base.Add(time.Duration(i) * step),
			Amount: amount,
		}
	}
	return points
}

func runningRange(base time.Time, from, to time.Duration) timeRange {
	return timeRange{start: base.Add(from), end: base.Add(to)}
}

func TestCalculateBalanceStats(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rate := func(v float64) *float64 { return &v }

	tests := []struct {
		name             string
		points           []BalanceDataPoint
		running          []timeRange
		wantRate         *float64
		wantSpanHours    float64
		wantRunningHours float64
		wantSampleCount  int
		wantEnoughSample bool
	}{
		{
			name:             "no points",
			points:           nil,
			wantRate:         nil,
			wantSpanHours:    0,
			wantRunningHours: 0,
			wantSampleCount:  0,
			wantEnoughSample: false,
		},
		{
			name:             "single point",
			points:           balancePoints(base, time.Hour, 100),
			wantRate:         nil,
			wantSpanHours:    0,
			wantRunningHours: 0,
			wantSampleCount:  1,
			wantEnoughSample: false,
		},
		{
			name:             "short running window is not enough even with a drop",
			points:           balancePoints(base, 30*time.Minute, 151.07, 151.07, 151.07, 151.05),
			running:          []timeRange{runningRange(base, 0, 2*time.Hour)},
			wantRate:         nil,
			wantSpanHours:    1.5,
			wantRunningHours: 1.5,
			wantSampleCount:  4,
			wantEnoughSample: false,
		},
		{
			name:             "unchanged intervals count toward running time",
			points:           balancePoints(base, time.Hour, 100, 100, 100, 99, 99, 98, 98),
			running:          []timeRange{runningRange(base, 0, 6*time.Hour)},
			wantRate:         rate(1.0 / 3.0),
			wantSpanHours:    6,
			wantRunningHours: 6,
			wantSampleCount:  7,
			wantEnoughSample: true,
		},
		{
			name:             "recharge does not offset consumption",
			points:           balancePoints(base, 3*time.Hour, 100, 99, 109, 108),
			running:          []timeRange{runningRange(base, 0, 9*time.Hour)},
			wantRate:         rate(2.0 / 9.0),
			wantSpanHours:    9,
			wantRunningHours: 9,
			wantSampleCount:  4,
			wantEnoughSample: true,
		},
		{
			name:             "flat balance has no rate",
			points:           balancePoints(base, time.Hour, 100, 100, 100, 100, 100, 100, 100),
			running:          []timeRange{runningRange(base, 0, 6*time.Hour)},
			wantRate:         nil,
			wantSpanHours:    6,
			wantRunningHours: 6,
			wantSampleCount:  7,
			wantEnoughSample: true,
		},
		{
			name:             "exactly at the minimum running time",
			points:           balancePoints(base, 6*time.Hour, 100, 94),
			running:          []timeRange{runningRange(base, 0, 6*time.Hour)},
			wantRate:         rate(1),
			wantSpanHours:    6,
			wantRunningHours: 6,
			wantSampleCount:  2,
			wantEnoughSample: true,
		},
		{
			name:             "drops outside instance uptime are ignored",
			points:           balancePoints(base, time.Hour, 100, 100, 100, 100, 100, 100, 100, 50),
			running:          []timeRange{runningRange(base, 0, 6*time.Hour)},
			wantRate:         nil,
			wantSpanHours:    7,
			wantRunningHours: 6,
			wantSampleCount:  8,
			wantEnoughSample: true,
		},
		{
			name: "only running intervals contribute to the rate",
			points: balancePoints(base, time.Hour,
				100, 99, 99, 98, 98, 97, 97, 50, 50),
			running:          []timeRange{runningRange(base, 0, 6*time.Hour)},
			wantRate:         rate(3.0 / 6.0),
			wantSpanHours:    8,
			wantRunningHours: 6,
			wantSampleCount:  9,
			wantEnoughSample: true,
		},
		{
			name:             "no instance uptime means no rate",
			points:           balancePoints(base, time.Hour, 100, 90, 80, 70, 60, 50, 40),
			running:          nil,
			wantRate:         nil,
			wantSpanHours:    6,
			wantRunningHours: 0,
			wantSampleCount:  7,
			wantEnoughSample: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBalanceStats(tt.points, tt.running)

			if got.SampleCount != tt.wantSampleCount {
				t.Errorf("SampleCount = %d, want %d", got.SampleCount, tt.wantSampleCount)
			}
			if math.Abs(got.SpanHours-tt.wantSpanHours) > 1e-9 {
				t.Errorf("SpanHours = %v, want %v", got.SpanHours, tt.wantSpanHours)
			}
			if math.Abs(got.RunningHours-tt.wantRunningHours) > 1e-9 {
				t.Errorf("RunningHours = %v, want %v", got.RunningHours, tt.wantRunningHours)
			}
			if got.HasEnoughSamples != tt.wantEnoughSample {
				t.Errorf("HasEnoughSamples = %v, want %v", got.HasEnoughSamples, tt.wantEnoughSample)
			}
			if got.MinRunningHours != BalanceStatsMinSpan.Hours() {
				t.Errorf("MinRunningHours = %v, want %v", got.MinRunningHours, BalanceStatsMinSpan.Hours())
			}

			switch {
			case tt.wantRate == nil && got.Rate != nil:
				t.Errorf("Rate = %v, want nil", *got.Rate)
			case tt.wantRate != nil && got.Rate == nil:
				t.Errorf("Rate = nil, want %v", *tt.wantRate)
			case tt.wantRate != nil && math.Abs(*got.Rate-*tt.wantRate) > 1e-9:
				t.Errorf("Rate = %v, want %v", *got.Rate, *tt.wantRate)
			}
		})
	}
}

func TestCalculateBalanceStatsSortsPoints(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := balancePoints(base, time.Hour, 100, 100, 100, 99, 99, 98, 98)
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}

	got := calculateBalanceStats(points, []timeRange{runningRange(base, 0, 6*time.Hour)})
	if got.Rate == nil {
		t.Fatal("Rate = nil, want 1/3")
	}
	if math.Abs(*got.Rate-1.0/3.0) > 1e-9 {
		t.Errorf("Rate = %v, want %v", *got.Rate, 1.0/3.0)
	}
}

func TestMergeTimeRanges(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	merged := mergeTimeRanges([]timeRange{
		runningRange(base, 4*time.Hour, 6*time.Hour),
		runningRange(base, 0, 2*time.Hour),
		runningRange(base, time.Hour, 5*time.Hour),
		{start: base.Add(8 * time.Hour), end: base.Add(8 * time.Hour)},
	})

	want := []timeRange{
		runningRange(base, 0, 6*time.Hour),
	}
	if len(merged) != len(want) {
		t.Fatalf("len(merged) = %d, want %d", len(merged), len(want))
	}
	for i := range want {
		if !merged[i].start.Equal(want[i].start) || !merged[i].end.Equal(want[i].end) {
			t.Errorf("merged[%d] = (%v, %v), want (%v, %v)",
				i, merged[i].start, merged[i].end, want[i].start, want[i].end)
		}
	}
}

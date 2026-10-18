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

func TestCalculateBalanceStats(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rate := func(v float64) *float64 { return &v }

	tests := []struct {
		name             string
		points           []BalanceDataPoint
		wantRate         *float64
		wantSpanHours    float64
		wantSampleCount  int
		wantEnoughSample bool
	}{
		{
			name:             "no points",
			points:           nil,
			wantRate:         nil,
			wantSpanHours:    0,
			wantSampleCount:  0,
			wantEnoughSample: false,
		},
		{
			name:             "single point",
			points:           balancePoints(base, time.Hour, 100),
			wantRate:         nil,
			wantSpanHours:    0,
			wantSampleCount:  1,
			wantEnoughSample: false,
		},
		{
			name: "short window is not enough even with a drop",
			points: balancePoints(base, 30*time.Minute,
				151.07, 151.07, 151.07, 151.05),
			wantRate:         nil,
			wantSpanHours:    1.5,
			wantSampleCount:  4,
			wantEnoughSample: false,
		},
		{
			name: "unchanged intervals count toward the window",
			points: balancePoints(base, time.Hour,
				100, 100, 100, 99, 99, 98, 98),
			wantRate:         rate(1.0 / 3.0),
			wantSpanHours:    6,
			wantSampleCount:  7,
			wantEnoughSample: true,
		},
		{
			name: "recharge does not offset consumption",
			points: balancePoints(base, 3*time.Hour,
				100, 99, 109, 108),
			wantRate:         rate(2.0 / 9.0),
			wantSpanHours:    9,
			wantSampleCount:  4,
			wantEnoughSample: true,
		},
		{
			name:             "flat balance has no rate",
			points:           balancePoints(base, time.Hour, 100, 100, 100, 100, 100, 100, 100),
			wantRate:         nil,
			wantSpanHours:    6,
			wantSampleCount:  7,
			wantEnoughSample: true,
		},
		{
			name:             "exactly at the minimum span",
			points:           balancePoints(base, 6*time.Hour, 100, 94),
			wantRate:         rate(1),
			wantSpanHours:    6,
			wantSampleCount:  2,
			wantEnoughSample: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBalanceStats(tt.points)

			if got.SampleCount != tt.wantSampleCount {
				t.Errorf("SampleCount = %d, want %d", got.SampleCount, tt.wantSampleCount)
			}
			if math.Abs(got.SpanHours-tt.wantSpanHours) > 1e-9 {
				t.Errorf("SpanHours = %v, want %v", got.SpanHours, tt.wantSpanHours)
			}
			if got.HasEnoughSamples != tt.wantEnoughSample {
				t.Errorf("HasEnoughSamples = %v, want %v", got.HasEnoughSamples, tt.wantEnoughSample)
			}
			if got.MinSpanHours != BalanceStatsMinSpan.Hours() {
				t.Errorf("MinSpanHours = %v, want %v", got.MinSpanHours, BalanceStatsMinSpan.Hours())
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

	got := calculateBalanceStats(points)
	if got.Rate == nil {
		t.Fatal("Rate = nil, want 1/3")
	}
	if math.Abs(*got.Rate-1.0/3.0) > 1e-9 {
		t.Errorf("Rate = %v, want %v", *got.Rate, 1.0/3.0)
	}
}

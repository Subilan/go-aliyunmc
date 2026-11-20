package monitors

import (
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
)

func TestResolveIdleReminderBefore(t *testing.T) {
	logger := log_util.NewNamedLogger("[test/auto-archive-idle] ", "auto-archive-idle-reminder-test")

	tests := []struct {
		name     string
		idleSec  int
		reminder int
		want     time.Duration
	}{
		{name: "关闭", idleSec: 1200, reminder: 0, want: 0},
		{name: "负数", idleSec: 1200, reminder: -1, want: 0},
		{name: "等于倒计时", idleSec: 1200, reminder: 1200, want: 0},
		{name: "大于倒计时", idleSec: 1200, reminder: 1800, want: 0},
		{name: "超过七天", idleSec: 2_000_000_000, reminder: 604_801, want: 0},
		{name: "合法", idleSec: 1200, reminder: 600, want: 10 * time.Minute},
		{name: "最小值", idleSec: 1200, reminder: 1, want: time.Second},
		{name: "正好七天", idleSec: 2_000_000_000, reminder: 604_800, want: 604_800 * time.Second},
		{name: "倒计时为零", idleSec: 0, reminder: 5, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveIdleReminderBefore(logger, tt.idleSec, tt.reminder); got != tt.want {
				t.Fatalf("resolveIdleReminderBefore(%d, %d) = %v, want %v", tt.idleSec, tt.reminder, got, tt.want)
			}
		})
	}
}

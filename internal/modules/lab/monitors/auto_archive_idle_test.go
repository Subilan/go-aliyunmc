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

func TestDecideIdleReminder(t *testing.T) {
	const reminderBefore = 10 * time.Minute

	running := func(remainingSeconds, idleSeconds int64) IdleCountdownState {
		return IdleCountdownState{Status: IdleCountdownRunning, RemainingSeconds: remainingSeconds, IdleSeconds: idleSeconds}
	}
	paused := func(remainingSeconds, idleSeconds int64) IdleCountdownState {
		return IdleCountdownState{Status: IdleCountdownPaused, RemainingSeconds: remainingSeconds, IdleSeconds: idleSeconds}
	}

	tests := []struct {
		name          string
		state         IdleCountdownState
		sent          bool
		wantSend      bool
		wantSent      bool
		wantIdleMin   int64
		wantRemainMin int64
	}{
		{
			name:          "达到阈值时发送",
			state:         running(600, 600),
			wantSend:      true,
			wantSent:      true,
			wantIdleMin:   10,
			wantRemainMin: 10,
		},
		{
			name:          "未到阈值",
			state:         running(601, 599),
			wantSent:      false,
			wantIdleMin:   0,
			wantRemainMin: 0,
		},
		{
			name:          "已经发送过不重复",
			state:         running(300, 900),
			sent:          true,
			wantSent:      true,
			wantIdleMin:   0,
			wantRemainMin: 0,
		},
		{
			name:     "新一轮倒计时重置发送状态",
			state:    running(1200, 0),
			sent:     true,
			wantSent: false,
		},
		{
			name:          "暂停时既不发也不重置",
			state:         paused(300, 900),
			sent:          true,
			wantSent:      true,
			wantIdleMin:   0,
			wantRemainMin: 0,
		},
		{
			name:     "倒计时结束后重置发送状态",
			state:    IdleCountdownState{Status: IdleCountdownInactive},
			sent:     true,
			wantSent: false,
		},
		{
			name:          "剩余为零不发送",
			state:         running(0, 1200),
			wantSent:      false,
			wantIdleMin:   0,
			wantRemainMin: 0,
		},
		{
			name:          "分钟数取整",
			state:         running(61, 61),
			wantSend:      true,
			wantSent:      true,
			wantIdleMin:   1,
			wantRemainMin: 2,
		},
		{
			name:          "提醒关闭时不发送",
			state:         running(60, 1140),
			wantSend:      false,
			wantSent:      false,
			wantIdleMin:   0,
			wantRemainMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := reminderBefore
			if tt.name == "提醒关闭时不发送" {
				before = 0
			}

			send, sent, data := decideIdleReminder(tt.state, before, tt.sent)
			if send != tt.wantSend {
				t.Errorf("send = %v, want %v", send, tt.wantSend)
			}
			if sent != tt.wantSent {
				t.Errorf("sent = %v, want %v", sent, tt.wantSent)
			}
			if data.IdleMinutes != tt.wantIdleMin {
				t.Errorf("IdleMinutes = %d, want %d", data.IdleMinutes, tt.wantIdleMin)
			}
			if data.RemainingMinutes != tt.wantRemainMin {
				t.Errorf("RemainingMinutes = %d, want %d", data.RemainingMinutes, tt.wantRemainMin)
			}
		})
	}
}

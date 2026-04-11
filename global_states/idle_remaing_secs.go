package global_states

import (
	"sync"
	"time"
)

// approxIdleRemainingSecs 是一个全局变量，用来近似表示距离自动归档流程触发还有多少秒。如果为 -1，表示没有正在进行的自动归档倒计时。
var approxIdleRemainingSecs int64 = -1
var approxIdleRemainingSecsMu sync.Mutex
var stopApproxIdleCountdownCh chan struct{}

// GetApproxIdleRemaningSecs 返回一个近似的距离自动归档流程触发还有多少秒的值。
func GetApproxIdleRemaningSecs() int64 {
	approxIdleRemainingSecsMu.Lock()
	defer approxIdleRemainingSecsMu.Unlock()
	return approxIdleRemainingSecs
}

// BeginApproxIdleCountdown 开始一个近似的自动归档倒计时，持续时间为 secs 秒。期间可以通过 GetApproxIdleRemaningSecs 获取剩余秒数的近似值。
func BeginApproxIdleCountdown(secs int64) {
	approxIdleRemainingSecsMu.Lock()
	approxIdleRemainingSecs = secs
	approxIdleRemainingSecsMu.Unlock()

	if stopApproxIdleCountdownCh != nil {
		close(stopApproxIdleCountdownCh)
	}
	stopApproxIdleCountdownCh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				approxIdleRemainingSecsMu.Lock()
				approxIdleRemainingSecs--
				if approxIdleRemainingSecs <= 0 {
					approxIdleRemainingSecs = 0
					approxIdleRemainingSecsMu.Unlock()
					return
				}
				approxIdleRemainingSecsMu.Unlock()
			case <-stopApproxIdleCountdownCh:
				return
			}
		}
	}()
}

// ResetApproxIdleCountdown 停止自动归档倒计时，并将剩余秒数重置为 -1。
func ResetApproxIdleCountdown() {
	approxIdleRemainingSecsMu.Lock()
	defer approxIdleRemainingSecsMu.Unlock()

	if stopApproxIdleCountdownCh != nil {
		close(stopApproxIdleCountdownCh)
		stopApproxIdleCountdownCh = nil
	}
	approxIdleRemainingSecs = -1
}

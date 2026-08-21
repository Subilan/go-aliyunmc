package global_states

import (
	"time"

	"github.com/Subilan/go-aliyunmc/states"
)

// SpotInterruptionState 描述抢占式实例回收保护状态。
// 当检测到当前活跃实例进入待回收状态（OperationLocks 中的 LockReason 为 Recycling）时，
// 系统会保护性停机 Minecraft 服务器并阻止再次启动，直到实例重新恢复运行。
type SpotInterruptionState struct {
	// Active 表示是否正处于抢占式实例回收保护状态。
	Active bool `json:"active"`
	// InstanceId 是处于待回收状态的实例 ID。
	InstanceId string `json:"instanceId"`
	// TriggeredAt 是检测到回收通知的时间。
	TriggeredAt time.Time `json:"triggeredAt"`
}

// spotInterruptionStore 基于 HubbedStore 承载回收保护状态，既支持原子读写，
// 也支持 SSE 订阅广播（由 state_routes 的 /state/watch/spot-interruption 使用）。
var spotInterruptionStore = states.NewHubbedStore[SpotInterruptionState]()

// 初始化时发布一次空状态，确保 SSE 订阅方连接后能立即获得有效快照（UpdatedAt 非零）。
func init() {
	spotInterruptionStore.ForceStore(SpotInterruptionState{}, nil)
}

// SetSpotInterruption 进入抢占式实例回收保护状态。
func SetSpotInterruption(instanceId string) {
	spotInterruptionStore.ForceStore(SpotInterruptionState{
		Active:      true,
		InstanceId:  instanceId,
		TriggeredAt: time.Now(),
	}, nil)
}

// ClearSpotInterruption 解除抢占式实例回收保护状态。
func ClearSpotInterruption() {
	spotInterruptionStore.ForceStore(SpotInterruptionState{}, nil)
}

// GetSpotInterruption 返回当前的抢占式实例回收保护状态。
func GetSpotInterruption() SpotInterruptionState {
	return spotInterruptionStore.Snapshot().Value
}

// IsSpotInterruptionActive 返回当前是否正处于抢占式实例回收保护状态。
func IsSpotInterruptionActive() bool {
	return GetSpotInterruption().Active
}

// SpotInterruptionStore 返回回收保护状态的 HubbedStore，供 SSE watch handler 订阅快照与更新。
func SpotInterruptionStore() *states.HubbedStore[SpotInterruptionState] {
	return spotInterruptionStore
}

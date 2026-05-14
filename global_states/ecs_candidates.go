package global_states

import (
	"github.com/Subilan/go-aliyunmc/states"
	"sync"
)

var currentEcsCandidates = []states.EcsCandidate{} // 默认为空slice
var currentEcsCandidatesMu sync.RWMutex

// GetCurrentEcsCandidates 获取当前的ECS候选列表。你可以认为这个函数不会返回 nil。
func GetCurrentEcsCandidates() []states.EcsCandidate {
	currentEcsCandidatesMu.RLock()
	defer currentEcsCandidatesMu.RUnlock()
	return currentEcsCandidates
}

// SetCurrentEcsCandidates 设置当前的ECS候选列表。如果传入 nil 则会被转换为一个空的slice。
func SetCurrentEcsCandidates(candidates []states.EcsCandidate) {
	if candidates == nil {
		candidates = []states.EcsCandidate{}
	}
	currentEcsCandidatesMu.Lock()
	defer currentEcsCandidatesMu.Unlock()
	currentEcsCandidates = candidates
}
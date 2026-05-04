package states

import "fmt"

var (
	HSKeyInstanceStatus   = "instance_status_monitor"
	HSKeyServerStatus     = "server_status_monitor"
	HSKeyBestEcsCandidate = "best_ecs_candidate_monitor"
)

type ServerStatusState struct {
	Online      bool   `json:"online"`
	PlayerCount int64  `json:"playerCount"`
}

type EcsCandidate struct {
	InstanceType string  `json:"instanceType"`
	Memory       int     `json:"memory"`
	CpuCoreCount int     `json:"cpuCoreCount"`
	ZoneId       string  `json:"zoneId"`
	TradePrice   float32 `json:"tradePrice"`
}

func (candidate EcsCandidate) String() string {
	return fmt.Sprintf("InstanceType: %s, Memory: %d, CpuCoreCount: %d, ZoneId: %s, TradePrice: %.2f",
		candidate.InstanceType, candidate.Memory, candidate.CpuCoreCount, candidate.ZoneId, candidate.TradePrice)
}

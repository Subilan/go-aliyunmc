package consts

type InstanceChargeSortBy string

const (
	ICSortByNone         InstanceChargeSortBy = ""
	ICSortByCpuCoreCount InstanceChargeSortBy = "cpuCoreCount"
	ICSortByMemorySize   InstanceChargeSortBy = "memorySize"
	ICSortByTradePrice   InstanceChargeSortBy = "tradePrice"
)

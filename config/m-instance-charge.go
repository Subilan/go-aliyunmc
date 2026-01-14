package config

import (
	"time"
)

// InstanceCharge 包含对 monitors.InstanceCharge 的相关配置
//
// MemChoices 和 CpuCoreCountChoices 会共同决定在刷新相关数据时发送 DescribeAvailableResource 的个数。具体来说，按照排列组合将发送
//
//	len(MemChoices) × len(CpuCoreCountChoices)
//
// 次请求以获取所有可能的组合对应的实例。
type InstanceCharge struct {
	// Interval 是刷新的时间间隔，单位秒
	Interval int `toml:"interval" validate:"required,gte=1"`

	// RetryInterval 是重试的时间间隔，单位秒
	RetryInterval int `toml:"retry_interval" validate:"required,gte=1"`

	// Timeout 是刷新的超时时间
	Timeout int `toml:"timeout" validate:"required,gte=1"`

	// MemChoices 表示可接受的实例内存大小，单位 GiB。可以选中多个实例内存大小离散值。
	MemChoices []int `toml:"mem_choices" validate:"required,dive,gte=1"`

	// CpuCoreCountChoices 表示可接受的实例虚拟 CPU 核数。可以选中多个实例核数离散值。
	CpuCoreCountChoices []int `toml:"cpu_core_count_choices" validate:"required,dive,gte=1"`

	// Filters 包含了对获得的实例的筛选配置
	Filters InstanceChargeFilters `toml:"filters" validate:"required"`

	// CacheFile 是刷新数据的缓存文件名，必须以 .json 结尾。系统刚启动时将先使用该文件记录的信息。
	CacheFile string `toml:"cache_file" validate:"required,endswith=.json"`
}

func (i InstanceCharge) TimeoutDuration() time.Duration {
	return time.Duration(i.Timeout) * time.Second
}

func (i InstanceCharge) IntervalDuration() time.Duration {
	return time.Duration(i.Interval) * time.Second
}

func (i InstanceCharge) RetryIntervalDuration() time.Duration {
	return time.Duration(i.RetryInterval) * time.Second
}

// InstanceChargeFilters 包含了对获取到的实例设置的筛选条件。
//
// InstanceTypeExclusion 主要用于过滤一些性能不佳或者不适合运行 Minecraft 服务器的实例类型，例如共享性实例（ecs.e*）的性能不佳，大数据型实例（ecs.d*）对 Minecraft 服务的运行意义不大。示例：
//
//	^ecs\\.(e|s6|xn4|n4|mn4|e4|t|d).*$
type InstanceChargeFilters struct {
	// MaxTradePrice 表示相关实例的最大交易价格，单位 CNY。超过（>）该交易价格的实例会被筛除。
	MaxTradePrice float32 `toml:"max_trade_price" validate:"gte=0"`

	// InstanceTypeExclusion 是一个正则表达式，表示对实例规格（如 ecs.g6.xlarge）的筛选条件。
	InstanceTypeExclusion string `toml:"instance_type_exclusion"`
}

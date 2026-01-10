package config

import (
	"time"
)

type InstanceCharge struct {
	Interval          int                   `toml:"interval" validate:"required,gte=1"`
	RetryInterval     int                   `toml:"retry_interval" validate:"required,gte=1"`
	Timeout           int                   `toml:"timeout" validate:"required,gte=1"`
	MemRange          []int                 `toml:"mem_range" validate:"required,posRange"`
	CpuCoreCountRange []int                 `toml:"cpu_core_count_range" validate:"required,posRange"`
	Filters           InstanceChargeFilters `toml:"filters" validate:"required"`
}

func (i InstanceCharge) MemIntRange() IntRange {
	return IntRange{i.MemRange[0], i.MemRange[1]}
}

func (i InstanceCharge) CpuCoreCountIntRange() IntRange {
	return IntRange{i.CpuCoreCountRange[0], i.CpuCoreCountRange[1]}
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

type InstanceChargeFilters struct {
	MaxTradePrice float32 `toml:"max_trade_price" validate:"gte=0"`
}

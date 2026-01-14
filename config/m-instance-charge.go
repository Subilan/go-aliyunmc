package config

import (
	"time"
)

type InstanceCharge struct {
	Interval            int                   `toml:"interval" validate:"required,gte=1"`
	RetryInterval       int                   `toml:"retry_interval" validate:"required,gte=1"`
	Timeout             int                   `toml:"timeout" validate:"required,gte=1"`
	MemChoices          []int                 `toml:"mem_choices" validate:"required,dive,gte=1"`
	CpuCoreCountChoices []int                 `toml:"cpu_core_count_choices" validate:"required,dive,gte=1"`
	Filters             InstanceChargeFilters `toml:"filters" validate:"required"`
	CacheFile           string                `toml:"cache_file" validate:"required"`
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
	MaxTradePrice         float32 `toml:"max_trade_price" validate:"gte=0"`
	InstanceTypeExclusion string  `toml:"instance_type_exclusion"`
}

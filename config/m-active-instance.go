package config

import "time"

// ActiveInstance 是对 monitors.ActiveInstance 的配置
type ActiveInstance struct {
	// Interval 是实例状态刷新的间隔，单位秒
	Interval int `toml:"interval" validate:"required,gte=1" comment:"刷新间隔，单位秒"`
}

func (a ActiveInstance) IntervalDuration() time.Duration {
	return time.Duration(a.Interval) * time.Second
}

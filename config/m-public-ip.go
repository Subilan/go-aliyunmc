package config

import "time"

// PublicIP 包含了 monitors.PublicIP 的相关配置
type PublicIP struct {
	// Interval 表示尝试获取实例状态的间隔，单位秒
	Interval int `toml:"interval" validate:"required,gte=1" comment:"刷新间隔，单位秒"`

	// Timeout 表示获取实例状态的超时时间，单位秒
	Timeout int `toml:"timeout" validate:"required,gte=1" comment:"超时时间，单位秒"`
}

func (p PublicIP) TimeoutDuration() time.Duration {
	return time.Duration(p.Timeout) * time.Second
}

func (p PublicIP) IntervalDuration() time.Duration {
	return time.Duration(p.Interval) * time.Second
}

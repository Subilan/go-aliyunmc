package config

import "time"

// StartInstance 包含了 monitors.StartActiveInstanceWhenReady 的相关配置
type StartInstance struct {
	// Interval 表示尝试获取实例状态的间隔，单位秒
	Interval int `toml:"interval" validate:"required,gte=1" comment:"刷新间隔，单位秒"`

	// Timeout 表示获取实例状态的超时时间，单位秒
	Timeout int `toml:"timeout" validate:"required,gte=1" comment:"超时时间，单位秒"`
}

func (s StartInstance) IntervalDuration() time.Duration {
	return time.Duration(s.Interval) * time.Second
}

func (s StartInstance) TimeoutDuration() time.Duration {
	return time.Duration(s.Timeout) * time.Second
}

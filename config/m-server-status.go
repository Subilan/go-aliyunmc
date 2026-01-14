package config

import "time"

// ServerStatus 包含了 monitors.ServerStatus 的相关配置
type ServerStatus struct {
	// Interval 表示尝试获取服务器状态的间隔，单位秒
	Interval int `toml:"interval" validate:"required,gte=1"`

	// Timeout 表示获取服务器状态的超时时间，单位秒
	Timeout int `toml:"timeout" validate:"required,gte=1"`
}

func (s ServerStatus) IntervalDuration() time.Duration {
	return time.Duration(s.Interval) * time.Second
}

func (s ServerStatus) TimeoutDuration() time.Duration {
	return time.Duration(s.Timeout) * time.Second
}

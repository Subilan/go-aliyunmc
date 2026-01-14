package config

import "time"

// Backup 是 monitors.Backup 的相关配置。
type Backup struct {
	// Interval 是备份的间隔，单位为秒。此间隔适用于备份正常进行时。
	Interval int `toml:"interval" validate:"required,gte=1"`

	// RetryInterval 是备份的重试间隔，单位为秒。当备份失败时，将优先采用此间隔进行重试。
	//
	// 如果实例没有运行，也会使用此间隔以快速发现正在运行的实例所需要的备份任务。
	//
	// 建议满足 RetryInterval < Interval 且 RetryInterval 设置为较短间隔，如 60s
	RetryInterval int `toml:"retry_interval" validate:"required,gte=1"`

	// Timeout 是备份的超时时间，单位为秒。如果超过此时间，备份会被中止且认为失败。
	Timeout int `toml:"timeout" validate:"required,gte=1"`
}

func (b Backup) IntervalDuration() time.Duration {
	return time.Duration(b.Interval) * time.Second
}

func (b Backup) RetryIntervalDuration() time.Duration {
	return time.Duration(b.RetryInterval) * time.Second
}

func (b Backup) TimeoutDuration() time.Duration {
	return time.Duration(b.Timeout) * time.Second
}

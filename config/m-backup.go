package config

import "time"

type Backup struct {
	Interval      int `toml:"interval" validate:"required,gte=1"`
	RetryInterval int `toml:"retry_interval" validate:"required,gte=1"`
	Timeout       int `toml:"timeout" validate:"required,gte=1"`
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

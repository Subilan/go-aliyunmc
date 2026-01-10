package config

import "time"

type StartInstance struct {
	Interval int `toml:"interval" validate:"required,gte=1"`
	Timeout  int `toml:"timeout" validate:"required,gte=1"`
}

func (s StartInstance) IntervalDuration() time.Duration {
	return time.Duration(s.Interval) * time.Second
}

func (s StartInstance) TimeoutDuration() time.Duration {
	return time.Duration(s.Timeout) * time.Second
}

package config

import "time"

type ServerStatus struct {
	Interval int `toml:"interval" validate:"required,gte=1"`
	Timeout  int `toml:"timeout" validate:"required,gte=1"`
}

func (s ServerStatus) IntervalDuration() time.Duration {
	return time.Duration(s.Interval) * time.Second
}

func (s ServerStatus) TimeoutDuration() time.Duration {
	return time.Duration(s.Timeout) * time.Second
}

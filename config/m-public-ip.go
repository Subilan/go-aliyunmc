package config

import "time"

type PublicIp struct {
	Interval int `toml:"interval" validate:"required,gte=1"`
	Timeout  int `toml:"timeout" validate:"required,gte=1"`
}

func (p PublicIp) TimeoutDuration() time.Duration {
	return time.Duration(p.Timeout) * time.Second
}

func (p PublicIp) IntervalDuration() time.Duration {
	return time.Duration(p.Interval) * time.Second
}

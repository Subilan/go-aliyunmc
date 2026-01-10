package config

import "time"

type ActiveInstance struct {
	Interval int `toml:"interval" validate:"required,gte=1"`
}

func (a ActiveInstance) IntervalDuration() time.Duration {
	return time.Duration(a.Interval) * time.Second
}

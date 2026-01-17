package config

import "time"

type BssSync struct {
	Interval    int       `toml:"interval" validate:"required,gte=1"`
	Timeout     int       `toml:"timeout" validate:"required,gte=1"`
	InitialTime time.Time `toml:"initial_time" validate:"required"`
}

func (b *BssSync) IntervalDuration() time.Duration {
	return time.Duration(b.Interval) * time.Second
}

func (b *BssSync) TimeoutDuration() time.Duration {
	return time.Duration(b.Timeout) * time.Second
}

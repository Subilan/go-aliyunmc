package config

import "time"

type EmptyServer struct {
	EmptyTimeout int `toml:"empty_timeout" validate:"required,gte=1"`
}

func (e EmptyServer) EmptyTimeoutDuration() time.Duration {
	return time.Duration(e.EmptyTimeout) * time.Second
}

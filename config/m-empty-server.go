package config

import "time"

// EmptyServer 包含了 monitors.EmptyServer 的相关配置
type EmptyServer struct {
	// EmptyTimeout 表示服务器的空转超时时间，单位秒
	//
	// 当 monitors.EmptyServer 发现服务器无玩家在线的状态持续超过此时间后，将停止服务器、归档并删除实例。
	EmptyTimeout int `toml:"empty_timeout" validate:"required,gte=1" comment:"服务器空转超时时间，单位秒。超过此时间，服务器会被关闭、归档并删除"`
}

func (e EmptyServer) EmptyTimeoutDuration() time.Duration {
	return time.Duration(e.EmptyTimeout) * time.Second
}

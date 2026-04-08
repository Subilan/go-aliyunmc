package monitors

import "go-aliyunmc/utils"

type ServerStatusMonitorConfig struct {
	PollIntervalSec int `toml:"poll_interval_sec" validate:"required,min=1" comment:"服务器状态轮询间隔（秒）"`
}

type InstanceStatusMonitorConfig struct {
	PollIntervalSec int `toml:"poll_interval_sec" validate:"required,min=1" comment:"实例状态轮询间隔（秒）"`
}

var ServerStatusC ServerStatusMonitorConfig
var InstanceStatusC InstanceStatusMonitorConfig

func MustLoadConfig() {
	utils.MustBindConfig(&ServerStatusC, "monitor-server")
	utils.MustBindConfig(&InstanceStatusC, "monitor-instance")
}

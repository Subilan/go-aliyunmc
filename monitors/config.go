package monitors

import "go-aliyunmc/utils"

type ServerStatusMonitorConfig struct {
	PollIntervalSec int `toml:"poll_interval_sec" validate:"required,min=1" comment:"服务器状态轮询间隔（秒）"`
}

type InstanceStatusMonitorConfig struct {
	PollIntervalSec int `toml:"poll_interval_sec" validate:"required,min=1" comment:"实例状态轮询间隔（秒）"`
}

type FileConfig struct {
	RemotePath      string `toml:"remote_path" validate:"required" comment:"远端文件路径"`
	LocalPath       string `toml:"local_path" validate:"required" comment:"本地文件相对路径（相对于 LocalCacheRoot）"`
	PollIntervalSec int    `toml:"poll_interval_sec" validate:"required,min=1" comment:"轮询间隔（秒）"`
}

type FileSyncMonitorConfig struct {
	LocalCacheRoot string       `toml:"local_cache_root" validate:"required" comment:"本地缓存根目录"`
	Files          []FileConfig `toml:"files" validate:"required,min=1" comment:"文件同步配置列表"`
}

var ServerStatusC ServerStatusMonitorConfig
var InstanceStatusC InstanceStatusMonitorConfig
var FileSyncC FileSyncMonitorConfig

func MustLoadConfig() {
	utils.MustBindConfig(&ServerStatusC, "monitor-server")
	utils.MustBindConfig(&InstanceStatusC, "monitor-instance")
	utils.MustBindConfig(&FileSyncC, "monitor-file-sync")
}

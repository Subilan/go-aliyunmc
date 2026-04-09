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

type AutoArchiveIdleMonitorConfig struct {
	Enabled                 bool `toml:"enabled" comment:"是否启用空服自动归档回收监控"`
	IdleCountdownSec        int  `toml:"idle_countdown_sec" validate:"required,min=1" comment:"空服触发回收的倒计时（秒）"`
	StopWaitTimeoutSec      int  `toml:"stop_wait_timeout_sec" validate:"required,min=1" comment:"发送stop后等待离线的最长时间（秒）"`
	OfflineCheckIntervalSec int  `toml:"offline_check_interval_sec" validate:"required,min=1" comment:"等待离线时的状态检查间隔（秒）"`
	DeleteIgnoreNonExistent bool `toml:"delete_ignore_non_existent" comment:"删除active instance时是否忽略不存在错误"`
}

var ServerStatusC ServerStatusMonitorConfig
var InstanceStatusC InstanceStatusMonitorConfig
var FileSyncC FileSyncMonitorConfig
var AutoArchiveIdleC AutoArchiveIdleMonitorConfig

func MustLoadConfig() {
	utils.MustBindConfig(&ServerStatusC, "monitor-server")
	utils.MustBindConfig(&InstanceStatusC, "monitor-instance")
	utils.MustBindConfig(&FileSyncC, "monitor-file-sync")
	utils.MustBindConfig(&AutoArchiveIdleC, "monitor-auto-archive-idle")
}

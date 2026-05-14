package monitors

import "github.com/Subilan/go-aliyunmc/utils"

type ServerStatusMonitorConfig struct {
	PollIntervalSec int `toml:"poll_interval_sec" validate:"required,min=1" comment:"服务器状态轮询间隔（秒）"`
}

type InstanceStatusMonitorConfig struct {
	PollIntervalSec int `toml:"poll_interval_sec" validate:"required,min=1" comment:"实例状态轮询间隔（秒）"`
}

type FileConfig struct {
	RemotePath      string `toml:"remote_path" validate:"required" comment:"远端文件路径"`
	Id              string `toml:"id" validate:"min=1,omitempty" comment:"用于唯一标识该文件，如果不填则使用LocalPath"`
	LocalPath       string `toml:"local_path" validate:"required" comment:"本地文件相对路径（相对于 LocalCacheRoot）"`
	PollIntervalSec int    `toml:"poll_interval_sec" validate:"required,min=1" comment:"轮询间隔（秒）"`
}

type DirConfig struct {
	RemotePath      string `toml:"remote_path" validate:"required" comment:"远端目录路径"`
	LocalPath       string `toml:"local_path" validate:"required" comment:"本地目录相对路径（相对于 LocalCacheRoot）"`
	PollIntervalSec int    `toml:"poll_interval_sec" validate:"required,min=1" comment:"轮询间隔（秒）"`
}

type FileSyncMonitorConfig struct {
	LocalCacheRoot string       `toml:"local_cache_root" validate:"required" comment:"本地缓存根目录"`
	Files          []FileConfig `toml:"files" validate:"omitempty" comment:"文件同步配置列表"`
	Dirs           []DirConfig  `toml:"dirs" validate:"omitempty" comment:"目录同步配置列表"`
}

type AutoArchiveIdleMonitorConfig struct {
	Enabled                 bool `toml:"enabled" comment:"是否启用空服自动归档回收监控"`
	IdleCountdownSec        int  `toml:"idle_countdown_sec" validate:"required,min=1" comment:"空服触发回收的倒计时（秒）"`
	StopWaitTimeoutSec      int  `toml:"stop_wait_timeout_sec" validate:"required,min=1" comment:"发送stop后等待离线的最长时间（秒）"`
	OfflineCheckIntervalSec int  `toml:"offline_check_interval_sec" validate:"required,min=1" comment:"等待离线时的状态检查间隔（秒）"`
	DeleteIgnoreNonExistent bool `toml:"delete_ignore_non_existent" comment:"删除active instance时是否忽略不存在错误"`
}

type BackupMonitorConfig struct {
	Enabled         bool `toml:"enabled" comment:"是否启用自动备份监控"`
	PollIntervalSec int  `toml:"poll_interval_sec" validate:"required,min=1" comment:"自动触发备份任务的轮询间隔（秒）"`
}

type BestEcsCandidateMonitorConfig struct {
	PollIntervalSec     int                   `toml:"poll_interval_sec" validate:"required,min=1" comment:"最佳ECS候选监控的轮询间隔（秒）"`
	MemChoices          []int                 `toml:"mem_choices" validate:"required,dive,gte=1" comment:"实例可接受的内存大小列表，单位GiB，请使用常见内存大小，如4、8、16"`
	CpuCoreCountChoices []int                 `toml:"cpu_core_count_choices" validate:"required,dive,gte=1" comment:"实例可接受的vCPU数量。请使用与实例内存比满足1:2、1:4、1:8的数值"`
	Filters             InstanceChargeFilters `toml:"filters" validate:"required"`
	CacheFile           string                `toml:"cache_file" validate:"required" comment:"本地缓存文件路径，必须是一个json文件"`
}

// InstanceChargeFilters 包含了对获取到的实例设置的筛选条件。
//
// InstanceTypeExclusion 主要用于过滤一些性能不佳或者不适合运行 Minecraft 服务器的实例类型，例如共享性实例（ecs.e*）的性能不佳，大数据型实例（ecs.d*）对 Minecraft 服务的运行意义不大。示例：
//
//	^ecs\\.(e|s6|xn4|n4|mn4|e4|t|d).*$
type InstanceChargeFilters struct {
	// MaxTradePrice 表示相关实例的最大交易价格，单位 CNY。超过（>）该交易价格的实例会被筛除。
	MaxTradePrice float32 `toml:"max_trade_price" validate:"gte=0" comment:"实例最大可接受的交易价格，单位CNY。超过此价格的实例会被过滤"`

	// InstanceTypeExclusion 是一个正则表达式，表示对实例规格（如 ecs.g6.xlarge）的筛选条件。
	InstanceTypeExclusion string `toml:"instance_type_exclusion" comment:"正则表达式，表示对实例规格名（实例类型）的筛选，符合该正则表达式的实例会被过滤"`
}

// PlayerListSamplerConfig 定义了玩家列表采样器的配置。
type PlayerListSamplerConfig struct {
	// MaxDataPoints 表示玩家列表采样器在内存中最多存储的数据点数量。
	MaxDataPoints     int `toml:"max_data_points" validate:"required,min=1" comment:"内存中最多存储的数据点数量"`
	// SampleIntervalSec 采样间隔，单位为秒。
	SampleIntervalSec int `toml:"sample_interval_sec" validate:"required,min=1" comment:"采样间隔（秒）"`
}

// BalanceSamplerConfig 定义了余额采样器的配置。
type BalanceSamplerConfig struct {
	// MaxDataPoints 表示余额采样器在内存中最多存储的数据点数量。
	MaxDataPoints     int `toml:"max_data_points" validate:"required,min=1" comment:"内存中最多存储的数据点数量"`
	// SampleIntervalSec 采样间隔，单位为秒。
	SampleIntervalSec int `toml:"sample_interval_sec" validate:"required,min=1" comment:"采样间隔（秒）"`
}

var ServerStatusC ServerStatusMonitorConfig
var InstanceStatusC InstanceStatusMonitorConfig
var FileSyncC FileSyncMonitorConfig
var AutoArchiveIdleC AutoArchiveIdleMonitorConfig
var BackupC BackupMonitorConfig
var BestEcsCandidateC BestEcsCandidateMonitorConfig
var PlayerCountSamplerC PlayerListSamplerConfig
var BalanceSamplerC BalanceSamplerConfig

func init() {
	utils.MustBindConfigToml(&ServerStatusC, "monitor-server")
	utils.MustBindConfigToml(&InstanceStatusC, "monitor-instance")
	utils.MustBindConfigToml(&FileSyncC, "monitor-file-sync")
	utils.MustBindConfigToml(&AutoArchiveIdleC, "monitor-auto-archive-idle")
	utils.MustBindConfigToml(&BackupC, "monitor-backup")
	utils.MustBindConfigToml(&BestEcsCandidateC, "monitor-best-ecs-candidate")
	utils.MustBindConfigToml(&PlayerCountSamplerC, "sampler-player-count")
	utils.MustBindConfigToml(&BalanceSamplerC, "sampler-balance")
}

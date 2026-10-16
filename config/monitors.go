package config

// Monitors 对应 [monitors]。
type Monitors struct {
	Server           MonitorsServer           `toml:"server"`
	Instance         MonitorsInstance         `toml:"instance"`
	FileSync         MonitorsFileSync         `toml:"file_sync"`
	AutoArchiveIdle  MonitorsAutoArchiveIdle  `toml:"auto_archive_idle"`
	Backup           MonitorsBackup           `toml:"backup"`
	SpotInterruption MonitorsSpotInterruption `toml:"spot_interruption"`
	BestEcsCandidate MonitorsBestEcsCandidate `toml:"best_ecs_candidate"`
}

// MonitorsServer 对应 [monitors.server]。
type MonitorsServer struct {
	PollIntervalSec int `toml:"poll_interval_sec" default:"5" validate:"min=1" comment:"服务器状态轮询间隔（秒）"`
}

// MonitorsInstance 对应 [monitors.instance]。
type MonitorsInstance struct {
	PollIntervalSec int `toml:"poll_interval_sec" default:"5" validate:"min=1" comment:"实例状态轮询间隔（秒）"`
}

// MonitorsFileSyncFile 对应 [monitors.file_sync.files] 的元素。
type MonitorsFileSyncFile struct {
	RemotePath      string `toml:"remote_path" validate:"required" comment:"远端文件路径"`
	Id              string `toml:"id" validate:"omitempty,min=1" comment:"用于唯一标识该文件，如果不填则使用LocalPath"`
	LocalPath       string `toml:"local_path" validate:"required" comment:"本地文件相对路径（相对于 LocalCacheRoot）"`
	PollIntervalSec int    `toml:"poll_interval_sec" validate:"required,min=1" comment:"轮询间隔（秒）"`
}

// MonitorsFileSyncDir 对应 [monitors.file_sync.dirs] 的元素。
type MonitorsFileSyncDir struct {
	RemotePath      string `toml:"remote_path" validate:"required" comment:"远端目录路径"`
	LocalPath       string `toml:"local_path" validate:"required" comment:"本地目录相对路径（相对于 LocalCacheRoot）"`
	PollIntervalSec int    `toml:"poll_interval_sec" validate:"required,min=1" comment:"轮询间隔（秒）"`
}

// MonitorsFileSync 对应 [monitors.file_sync]。
type MonitorsFileSync struct {
	LocalCacheRoot string `toml:"local_cache_root" default:"remote_data_cache" comment:"本地缓存根目录"`

	Files []MonitorsFileSyncFile `toml:"files" validate:"dive" comment:"文件同步配置列表，每项形如 { remote_path = '...', local_path = '...', poll_interval_sec = 10 }"`
	Dirs  []MonitorsFileSyncDir  `toml:"dirs" validate:"dive" comment:"目录同步配置列表，每项形如 { remote_path = '...', local_path = '...', poll_interval_sec = 300 }"`
}

// MonitorsAutoArchiveIdle 对应 [monitors.auto_archive_idle]。
type MonitorsAutoArchiveIdle struct {
	Enabled                 bool `toml:"enabled" comment:"是否启用空服自动归档回收监控"`
	IdleCountdownSec        int  `toml:"idle_countdown_sec" default:"1200" validate:"min=1" comment:"空服触发回收的倒计时（秒）"`
	StopWaitTimeoutSec      int  `toml:"stop_wait_timeout_sec" default:"60" validate:"min=1" comment:"发送stop后等待离线的最长时间（秒）"`
	OfflineCheckIntervalSec int  `toml:"offline_check_interval_sec" default:"3" validate:"min=1" comment:"等待离线时的状态检查间隔（秒）"`
	DeleteIgnoreNonExistent bool `toml:"delete_ignore_non_existent" comment:"删除active instance时是否忽略不存在错误"`
}

// MonitorsBackup 对应 [monitors.backup]。
type MonitorsBackup struct {
	Enabled         bool `toml:"enabled" comment:"是否启用自动备份监控"`
	PollIntervalSec int  `toml:"poll_interval_sec" default:"600" validate:"min=1" comment:"自动触发备份任务的轮询间隔（秒）"`
}

// MonitorsSpotInterruption 对应 [monitors.spot_interruption]。
type MonitorsSpotInterruption struct {
	Enabled                 bool `toml:"enabled" comment:"是否启用抢占式实例回收监控"`
	PollIntervalSec         int  `toml:"poll_interval_sec" default:"30" validate:"min=1" comment:"回收状态轮询间隔（秒）"`
	StopWaitTimeoutSec      int  `toml:"stop_wait_timeout_sec" default:"90" validate:"min=1" comment:"发送stop后等待离线的最长时间（秒）"`
	OfflineCheckIntervalSec int  `toml:"offline_check_interval_sec" default:"3" validate:"min=1" comment:"等待离线时的状态检查间隔（秒）"`
}

// MonitorsBestEcsCandidateFilters 对应 [monitors.best_ecs_candidate.filters]，
// 包含了对获取到的实例设置的筛选条件。
//
// InstanceTypeExclusion 主要用于过滤一些性能不佳或者不适合运行 Minecraft 服务器的实例类型，例如共享性实例（ecs.e*）的性能不佳，大数据型实例（ecs.d*）对 Minecraft 服务的运行意义不大。示例：
//
//	^ecs\\.(e|s6|xn4|n4|mn4|e4|t|d).*$
type MonitorsBestEcsCandidateFilters struct {
	// MaxTradePrice 表示相关实例的最大交易价格，单位 CNY。超过（>）该交易价格的实例会被筛除。
	MaxTradePrice float32 `toml:"max_trade_price" default:"0.6" validate:"gte=0" comment:"实例最大可接受的交易价格，单位CNY。超过此价格的实例会被过滤"`

	// InstanceTypeExclusion 是一个正则表达式，表示对实例规格（如 ecs.g6.xlarge）的筛选条件。
	InstanceTypeExclusion string `toml:"instance_type_exclusion" comment:"正则表达式，表示对实例规格名（实例类型）的筛选，符合该正则表达式的实例会被过滤，例如 ^ecs\\\\.(e|s6|xn4|n4|mn4|e4|t|d).*$"`
}

// MonitorsBestEcsCandidate 对应 [monitors.best_ecs_candidate]。
type MonitorsBestEcsCandidate struct {
	PollIntervalSec int `toml:"poll_interval_sec" default:"600" validate:"min=1" comment:"最佳ECS候选监控的轮询间隔（秒）"`

	// MemChoices / CpuCoreCountChoices 不给默认值，理由同 TasksDeployVars.Packages。
	MemChoices          []int `toml:"mem_choices" validate:"required,dive,gte=1" comment:"实例可接受的内存大小列表，单位GiB，请使用常见内存大小，如 [8, 16]"`
	CpuCoreCountChoices []int `toml:"cpu_core_count_choices" validate:"required,dive,gte=1" comment:"实例可接受的vCPU数量，请使用与实例内存比满足1:2、1:4、1:8的数值，如 [4, 8]"`

	Filters MonitorsBestEcsCandidateFilters `toml:"filters"`

	CacheFile string `toml:"cache_file" default:"ecs_candidates.json" comment:"本地缓存文件路径，必须是一个json文件"`
}

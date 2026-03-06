package config

import "time"

// PlayerProfile 是 monitors.PlayerProfile 的相关配置。
type PlayerProfile struct {
	// Interval 是下载玩家数据的间隔，单位为秒。
	Interval int `toml:"interval" validate:"required,gte=1" comment:"刷新间隔，单位秒"`

	// Timeout 是下载玩家数据的超时时间，单位为秒。如果超过此时间，下载会被中止且认为失败。
	Timeout int `toml:"timeout" validate:"required,gte=1" comment:"下载超时时间，单位秒"`

	// EssentialsPath 是服务器上 Essentials 玩家数据的路径
	EssentialsPath string `toml:"essentials_path" validate:"required" comment:"服务器上 Essentials 玩家数据路径"`

	// StatsPath 是服务器上 Minecraft 统计数据的 JSON 路径
	StatsPath string `toml:"stats_path" validate:"required" comment:"服务器上 Minecraft 统计数据路径"`

	// EssentialsDir 是本地 Essentials 玩家数据的存储目录
	EssentialsDir string `toml:"essentials_dir" validate:"required" comment:"本地 Essentials 玩家数据存储目录"`

	// StatsDir 是本地 Minecraft 统计数据的存储目录
	StatsDir string `toml:"stats_dir" validate:"required" comment:"本地 Minecraft 统计数据存储目录"`
}

func (p PlayerProfile) IntervalDuration() time.Duration {
	return time.Duration(p.Interval) * time.Second
}

func (p PlayerProfile) TimeoutDuration() time.Duration {
	return time.Duration(p.Timeout) * time.Second
}

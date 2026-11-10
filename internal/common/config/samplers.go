package config

// Samplers 对应 [samplers]。
type Samplers struct {
	PlayerCount SamplersPlayerCount `toml:"player_count"`
	Balance     SamplersBalance     `toml:"balance"`
}

// SamplersPlayerCount 对应 [samplers.player_count]，定义了玩家列表采样器的配置。
type SamplersPlayerCount struct {
	// MaxDataPoints 表示玩家列表采样器在内存中最多存储的数据点数量。
	MaxDataPoints int `toml:"max_data_points" default:"100" comment:"内存中最多存储的数据点数量"`
	// SampleIntervalSec 采样间隔，单位为秒。
	SampleIntervalSec int `toml:"sample_interval_sec" default:"5" comment:"采样间隔（秒）"`
}

// SamplersBalance 对应 [samplers.balance]，定义了余额采样器的配置。
type SamplersBalance struct {
	// MaxDataPoints 表示余额采样器在内存中最多存储的数据点数量。
	MaxDataPoints int `toml:"max_data_points" default:"100" comment:"内存中最多存储的数据点数量"`
	// SampleIntervalSec 采样间隔，单位为秒。
	SampleIntervalSec int `toml:"sample_interval_sec" default:"300" comment:"采样间隔（秒）"`
}

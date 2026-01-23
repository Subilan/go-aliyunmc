package config

import "time"

type Whitelist struct {
	// Interval 表示尝试获取实例状态的间隔，单位秒
	Interval int `toml:"interval" validate:"required,gte=1" comment:"刷新间隔，单位秒"`

	// Timeout 表示获取实例状态的超时时间，单位秒
	Timeout int `toml:"timeout" validate:"required,gte=1" comment:"超时时间，单位秒"`

	// CacheFile 是白名单数据在本地缓存的文件名
	CacheFile string `toml:"cache_file" validate:"required,endswith=.json" comment:"白名单缓存文件名"`
}

func (w *Whitelist) IntervalDuration() time.Duration {
	return time.Duration(w.Interval) * time.Second
}

func (w *Whitelist) TimeoutDuration() time.Duration {
	return time.Duration(w.Timeout) * time.Second
}

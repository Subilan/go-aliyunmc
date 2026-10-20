// Package config 定义全部配置的结构体、加载流程与全局访问入口。
package config

import "sync/atomic"

// Config 是全部配置的聚合，对应 config/config.toml 的各顶层分节。
type Config struct {
	Base   Base   `toml:"base"`
	Aliyun Aliyun `toml:"aliyun"`
	Server Server `toml:"server"`
	SSH    SSH    `toml:"ssh"`

	UserRoutes UserRoutes `toml:"user_routes"`
	Tasks      Tasks      `toml:"tasks"`
	Monitors   Monitors   `toml:"monitors"`
	Samplers   Samplers   `toml:"samplers"`

	// Store 来自 config/store.toml，不参与 config.toml 的解析。
	Store Store `toml:"-"`
}

// current 是进程内唯一的一份配置，用 atomic 以便跨 goroutine 读取。
var current atomic.Pointer[Config]

// Get 返回进程内唯一的一份配置，加载之后只读，可在任意 goroutine 中调用。
func Get() *Config {
	cfg := current.Load()
	if cfg == nil {
		panic("config: 尚未调用 MustLoad/MustLoadStore")
	}
	return cfg
}

// TryGet 返回已安装的配置；尚未加载时返回 nil，供需要默认值兜底的库函数使用。
func TryGet() *Config {
	return current.Load()
}

// Set 安装配置。供测试与 migrate/create_user 这类特殊入口使用。
func Set(cfg *Config) {
	current.Store(cfg)
}

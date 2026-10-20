// Package config 持有本程序全部配置的结构体定义、加载与全局访问入口。
//
// 本包不 import 项目内的任何其他包。各功能包通过 config.Get() 读取配置，
// 自己只负责副作用（连库、建 client、载 Casbin、读私钥等）。
//
// 命名约定：
//
//   - 类型名 = 分节路径的驼峰形式：[monitors.file_sync] → MonitorsFileSync
//   - 聚合字段名 = 顶层分节名，结构体字段名 = 该节里的键
//   - config.Get().Monitors.FileSync.PollIntervalSec 对应
//     [monitors.file_sync] 里的 poll_interval_sec
//   - 每个类型上方注明它对应的 TOML 路径
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

// Package config 持有本程序全部配置的结构体定义、加载与全局访问入口。
//
// 所有配置结构体都集中在这里，而不是散落在各功能包里。原因是各功能包都要读取配置，
// 如果配置类型仍留在它们自己包里、由一个聚合包去引用，那么各包为了取配置而 import
// 聚合包就会形成循环。因此本包不 import 任何项目内的其他包，功能包统一通过
// config.Get() 读取，自己只负责副作用（连库、建 client、载 Casbin、读私钥等）。
//
// 命名约定——Go 没有 namespace，二十多个结构体平铺在一个包里必须靠命名区分：
//
//   - 类型名 = 分节路径的驼峰形式：[monitors.file_sync] → MonitorsFileSync
//   - 聚合字段名 = 顶层分节名，结构体字段名 = 该节里的键
//   - 于是 config.Get().Monitors.FileSync.PollIntervalSec 对应的就是
//     [monitors.file_sync] 里的 poll_interval_sec
//   - 每个类型上方都注明它对应的 TOML 路径
//
// 这条映射由 TestSectionPaths 守护。
package config

import "sync/atomic"

// Config 是全部配置的聚合，对应 configs/config.toml 的各顶层分节。
type Config struct {
	Base   Base   `toml:"base"`
	Aliyun Aliyun `toml:"aliyun"`
	Server Server `toml:"server"`
	Perms  Perms  `toml:"perms"`
	SSH    SSH    `toml:"ssh"`

	UserRoutes UserRoutes `toml:"user_routes"`
	Tasks      Tasks      `toml:"tasks"`
	Monitors   Monitors   `toml:"monitors"`
	Samplers   Samplers   `toml:"samplers"`

	// Store 来自独立的 configs/store.toml，不参与 config.toml 的解析。标成 "-" 之后
	// config.toml 里若出现 [store] 会被严格解码报错，不会形成第二条真相。
	Store Store `toml:"-"`
}

// current 是进程内唯一的一份配置。用 atomic 而非裸变量，是因为读取点分布在
// monitor 等的 goroutine 里，代价为零而能免除一切可见性疑虑。
var current atomic.Pointer[Config]

// Get 返回进程内唯一的一份配置。MustLoad/MustLoadStore 之后只读，
// 可以在任意 goroutine 中调用。
func Get() *Config {
	cfg := current.Load()
	if cfg == nil {
		panic("config: 尚未调用 MustLoad/MustLoadStore")
	}
	return cfg
}

// Set 安装配置。供测试与 migrate/create_user 这类特殊入口使用。
func Set(cfg *Config) {
	current.Store(cfg)
}

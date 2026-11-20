package config

// Bot 对应 [bot]：内网机器人接入配置。
type Bot struct {
	// Enabled 控制是否注册 /bot-v1 路由。
	Enabled bool `toml:"enabled" default:"false" comment:"是否启用内网机器人接口；关闭时不注册 /bot-v1 路由"`

	// RateLimitPerMinute 是按来源地址限制的每分钟请求数；0 表示不限制。
	RateLimitPerMinute int `toml:"rate_limit_per_minute" default:"120" validate:"min=0" comment:"每分钟允许的请求数；0 表示不限制"`

	// API 是后端调用机器人 HTTP 接口的连接配置。
	API BotAPI `toml:"api"`
}

// BotAPI 对应 [bot.api]：机器人进程暴露的本机 HTTP 接口。
// 机器人只监听回环地址，所以这里只需要配置端口。
type BotAPI struct {
	// Port 是机器人 HTTP 服务的端口。
	Port int `toml:"port" default:"8081" validate:"min=1,max=65535" comment:"机器人 HTTP 接口端口；后端固定连接 127.0.0.1"`
}

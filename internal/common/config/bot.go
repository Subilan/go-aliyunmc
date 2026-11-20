package config

// Bot 对应 [bot]：内网机器人接入配置。
type Bot struct {
	// Enabled 控制是否注册 /bot-v1 路由。
	Enabled bool `toml:"enabled" default:"false" comment:"是否启用内网机器人接口；关闭时不注册 /bot-v1 路由"`

	// RateLimitPerMinute 是按来源地址限制的每分钟请求数；0 表示不限制。
	RateLimitPerMinute int `toml:"rate_limit_per_minute" default:"120" validate:"min=0" comment:"每分钟允许的请求数；0 表示不限制"`
}

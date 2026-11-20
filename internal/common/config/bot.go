package config

// Bot 对应 [bot]：外部 QQ 机器人接入配置。
type Bot struct {
	// Enabled 控制是否注册 /bot-v1 路由。
	Enabled bool `toml:"enabled" default:"false" comment:"是否启用机器人接口；关闭时不注册 /bot-v1 路由"`

	// KeyID 是机器人密钥标识，请求头 X-Bot-Key-Id 必须与它一致。
	KeyID string `toml:"key_id" default:"default" validate:"required,max=64" comment:"机器人密钥标识，请求头 X-Bot-Key-Id 必须匹配"`

	// PublicKeyPath 是机器人公钥文件路径，用于验证请求签名。
	PublicKeyPath string `toml:"public_key_path,commented" validate:"required_if=Enabled true" comment:"机器人公钥文件路径（PEM，PKIX 格式），用于验证请求签名"`

	// Scopes 是授予机器人的权限范围。
	Scopes []string `toml:"scopes" validate:"unique,dive,oneof=read account admin" comment:"授予机器人的权限范围：read / account / admin"`

	// TimestampToleranceSec 是允许的请求时间偏差，同时决定 nonce 缓存时长。
	TimestampToleranceSec int `toml:"timestamp_tolerance_sec" default:"300" validate:"min=1,max=3600" comment:"允许的请求时间偏差（秒），同时决定 nonce 缓存时长"`

	// NonceCapacity 是防重放缓存容量。
	NonceCapacity int `toml:"nonce_capacity" default:"10000" validate:"min=1" comment:"防重放 nonce 缓存容量"`

	// MaxBodyBytes 是机器人请求体上限。
	MaxBodyBytes int64 `toml:"max_body_bytes" default:"1048576" validate:"min=1" comment:"机器人请求体上限（字节）"`

	// RateLimitPerMinute 是每分钟允许的请求数；0 表示不限制。
	RateLimitPerMinute int `toml:"rate_limit_per_minute" default:"120" validate:"min=0" comment:"每分钟允许的请求数；0 表示不限制"`
}

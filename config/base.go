package config

// BaseConfig 包含了系统本身的相关配置项目。
type BaseConfig struct {
	// Expose 是 HTTP 服务器的端口
	Expose int `toml:"expose" validate:"required"`

	// JwtSecret 是用于签名用户 JWT 令牌的私有字符串（密码）
	JwtSecret string `toml:"jwt_secret" validate:"required"`
}

package config

import "github.com/gin-contrib/cors"

// BaseConfig 包含了系统本身的相关配置项目。
type BaseConfig struct {
	// Expose 是 HTTP 服务器的端口
	Expose int `toml:"expose" validate:"required" comment:"HTTP服务器的监听端口"`

	// JwtSecret 是用于签名用户 JWT 令牌的私有字符串（密码）
	JwtSecret string `toml:"jwt_secret" validate:"required" comment:"用于为用户登录JWT令牌签名的私有字符串"`

	Cors CorsConfig `toml:"cors"`
}

func (b *BaseConfig) GetGinCorsConfig() cors.Config {
	return cors.Config{
		AllowMethods:     b.Cors.AllowMethods,
		AllowHeaders:     b.Cors.AllowHeaders,
		AllowOrigins:     b.Cors.AllowOrigins,
		AllowCredentials: b.Cors.AllowCredentials,
	}
}

// CorsConfig 是对 gin CORS 中间件的配置，字段解释见 https://pkg.go.dev/github.com/gin-contrib/cors#Config
type CorsConfig struct {
	AllowOrigins     []string `toml:"allow_origins" comment:"允许的源地址"`
	AllowMethods     []string `toml:"allow_methods" comment:"允许的请求方法"`
	AllowHeaders     []string `toml:"allow_headers" comment:"允许的请求头"`
	AllowCredentials bool     `toml:"allow_credentials" comment:"是否允许携带凭证信息"`
}

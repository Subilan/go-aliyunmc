package config

import "github.com/gin-contrib/cors"

// CorsConfig 是对 gin CORS 中间件的配置，字段解释见 https://pkg.go.dev/github.com/gin-contrib/cors#Config
type CorsConfig struct {
	AllowOrigins     []string `toml:"allow_origins" comment:"允许的源地址"`
	AllowMethods     []string `toml:"allow_methods" comment:"允许的请求方法"`
	AllowHeaders     []string `toml:"allow_headers" comment:"允许的请求头"`
	AllowCredentials bool     `toml:"allow_credentials" comment:"是否允许携带凭证信息"`
}

func (c *CorsConfig) GinCorsConfig() cors.Config {
	return cors.Config{
		AllowMethods:     c.AllowMethods,
		AllowHeaders:     c.AllowHeaders,
		AllowOrigins:     c.AllowOrigins,
		AllowCredentials: c.AllowCredentials,
	}
}
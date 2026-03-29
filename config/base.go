package config

type BaseConfig struct {
	// Expose 是 HTTP 服务器的端口
	Expose int `toml:"expose" validate:"required" comment:"HTTP服务器的监听端口"`

	// Cors 是针对 gin-contrib/cors 跨域中间件的设置，对应 cors.Config
	Cors CorsConfig `toml:"cors" validate:"required" comment:""`

	// Autotls 是针对 gin-gonic/autotls 中间件的设置
	Autotls AutotlsConfig `toml:"autotls" validate:"required" comment:""`

	// Session 是针对 gin-contrib/sessions 中间件的设置
	Session SessionConfig `toml:"session" validate:"required" comment:""`
}
package config

import "github.com/gin-contrib/cors"

// Base 对应 [base]：本服务自身（HTTP 服务器）的配置。
type Base struct {
	// Expose 是 HTTP 服务器的端口
	Expose int `toml:"expose" default:"45678" comment:"HTTP服务器的监听端口"`

	// Cors 是针对 gin-contrib/cors 跨域中间件的设置，对应 cors.Config
	Cors BaseCors `toml:"cors"`

	// TLS 是 HTTPS 配置，使用本地证书文件
	TLS BaseTLS `toml:"tls"`
}

// BaseCors 对应 [base.cors]，字段解释见 https://pkg.go.dev/github.com/gin-contrib/cors#Config
//
// 这几个列表故意不给默认值：空列表是合法且有意义的取值，给默认值会让显式写下的
// 空列表被静默替换。
type BaseCors struct {
	AllowOrigins     []string `toml:"allow_origins" comment:"允许的源地址，例如 ['https://example.com']"`
	AllowMethods     []string `toml:"allow_methods" comment:"允许的请求方法，例如 ['GET', 'POST']"`
	AllowHeaders     []string `toml:"allow_headers" comment:"允许的请求头，例如 ['Content-Type', 'Authorization']"`
	AllowCredentials bool     `toml:"allow_credentials" comment:"是否允许携带凭证信息"`
}

// GinCorsConfig 把本节的配置转成 gin-contrib/cors 需要的结构。
// 它紧跟着被转换的字段，改了字段就能在这里一眼看到对应关系。
func (c BaseCors) GinCorsConfig() cors.Config {
	return cors.Config{
		AllowMethods:     c.AllowMethods,
		AllowHeaders:     c.AllowHeaders,
		AllowOrigins:     c.AllowOrigins,
		AllowCredentials: c.AllowCredentials,
	}
}

// BaseTLS 对应 [base.tls]。证书路径只在启用 HTTPS 时才需要填写，因此示例里是注释掉的。
type BaseTLS struct {
	Enabled  bool   `toml:"enabled" comment:"是否启用HTTPS。如果启用，将覆盖Expose端口，使用443端口"`
	CertFile string `toml:"cert_file,commented" comment:"证书文件路径 (fullchain.cer)"`
	KeyFile  string `toml:"key_file,commented" comment:"私钥文件路径"`
	HttpPort int    `toml:"http_port" default:"80" comment:"HTTP明文端口，供内网服务访问（如DDNS）"`
}

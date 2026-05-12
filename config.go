package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
)

var C Config

type Config struct {
	// Expose 是 HTTP 服务器的端口
	Expose int `toml:"expose" validate:"required" comment:"HTTP服务器的监听端口"`

	// Cors 是针对 gin-contrib/cors 跨域中间件的设置，对应 cors.Config
	Cors CorsConfig `toml:"cors" validate:"required" comment:""`

	// TLS 是 HTTPS 配置，使用本地证书文件
	TLS TlsConfig `toml:"tls" validate:"required" comment:""`

	// Session 是针对 gin-contrib/sessions 中间件的设置
	Session SessionConfig `toml:"session" validate:"required" comment:""`
}

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

type TlsConfig struct {
	Enabled  bool   `toml:"enabled" comment:"是否启用HTTPS（生产模式）"`
	CertFile string `toml:"cert_file" comment:"证书文件路径 (fullchain.cer)"`
	KeyFile  string `toml:"key_file" comment:"私钥文件路径"`
	HttpPort int    `toml:"http_port" comment:"HTTP明文端口，供内网服务访问（如DDNS）" validate:"required_if=Enabled true"`
}

type SessionConfig struct {
	// KeyPairs 是密钥对列表。每个密钥对包含认证密钥和可选的加密密钥
	KeyPairs []SessionKeyPair `toml:"key_pairs" validate:"required,min=1,dive" comment:"密钥对列表，支持密钥轮换"`
}

type SessionKeyPair struct {
	// AuthKey 是认证密钥，必需
	AuthKey string `toml:"auth_key" validate:"required" comment:"认证密钥"`
	// EncKey 是加密密钥，可选
	EncKey string `toml:"enc_key" comment:"加密密钥"`
}

// GetSessionStore 返回 Session 配置对应的 sessions.Store
func (s *SessionConfig) GetSessionStore() sessions.Store {
	keyPairs := make([][]byte, 0, len(s.KeyPairs)*2)

	for _, pair := range s.KeyPairs {
		keyPairs = append(keyPairs, []byte(pair.AuthKey))
		if pair.EncKey != "" {
			keyPairs = append(keyPairs, []byte(pair.EncKey))
		}
	}

	return cookie.NewStore(keyPairs...)
}

package config

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
)

type SessionConfig struct {
	// KeyPairs 是密钥对列表，支持密钥轮换
	// 每个密钥对包含认证密钥和可选的加密密钥
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

package clients

import (
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/aliyun/credentials-go/credentials"
)

// MustGetAKCredential 参见 ShouldGetAKCredential，如果获取过程发生问题会 panic
func MustGetAKCredential() credentials.Credential {
	cr, err := ShouldGetAKCredential()

	if err != nil {
		panic(err)
	}

	return cr
}

// ShouldGetAKCredential 返回一个基于 config.toml 中配置的 AKID 和 AKSecret 形成的 credentials.Credential 以及可能的 error
func ShouldGetAKCredential() (credentials.Credential, error) {
	crConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(config.Cfg.Aliyun.AccessKeyId).
		SetAccessKeySecret(config.Cfg.Aliyun.AccessKeySecret)
	return credentials.NewCredential(crConfig)
}

package clients

import (
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/aliyun/credentials-go/credentials"
)

// MustGetAKCredential 是 ShouldGetAKCredential 的 Must 形式
func MustGetAKCredential() credentials.Credential {
	cr, err := ShouldGetAKCredential()

	if err != nil {
		panic(err)
	}

	return cr
}

// ShouldGetAKCredential 尝试返回一个基于 config.AliyunConfig 形成的 credentials.Credential
func ShouldGetAKCredential() (credentials.Credential, error) {
	crConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(config.Cfg.Aliyun.AccessKeyId).
		SetAccessKeySecret(config.Cfg.Aliyun.AccessKeySecret)
	return credentials.NewCredential(crConfig)
}

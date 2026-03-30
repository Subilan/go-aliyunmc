package aliyun

import (
	"github.com/aliyun/credentials-go/credentials"
)

// MustGetAKCredential 返回从配置中提取的 AK 凭证，如果无法提取则 panic
func MustGetAKCredential() credentials.Credential {
	crConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(C.AccessKeyId).
		SetAccessKeySecret(C.AccessKeySecret)
	cr, err := credentials.NewCredential(crConfig)

	if err != nil {
		panic(err)
	}

	return cr
}

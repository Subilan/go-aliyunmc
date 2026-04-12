package aliyun

import (
	"go-aliyunmc/log_util"

	"github.com/aliyun/credentials-go/credentials"
)

// MustInitialize 初始化各个阿里云client
func MustInitialize() {
	var err error

	OssClient = GetOssClient()
	EcsClient, err = ShouldCreateEcsClient()
	if err != nil {
		panic(err)
	}
	VpcClient, err = ShouldCreateVpcClient()
	if err != nil {
		panic(err)
	}
	BssClient, err = ShouldCreateBssClient()
	if err != nil {
		panic(err)
	}
}


// MustGetAKCredential 返回从配置中提取的 AK 凭证，如果无法提取则 panic
func MustGetAKCredential() credentials.Credential {
	crConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(C.AccessKeyId).
		SetAccessKeySecret(C.AccessKeySecret)
	cr, err := credentials.NewCredential(crConfig)

	if err != nil {
		log_util.Fatal("无法创建凭证数据")
	}

	return cr
}

package clients

import (
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// OssClient 是系统全局对象存储服务客户端
var OssClient *oss.Client

// GetOssClient 根据凭据创建一个对象存储服务客户端
func GetOssClient() *oss.Client {
	client := oss.NewClient(
		oss.LoadDefaultConfig().
			WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.Cfg.Aliyun.AccessKeyId, config.Cfg.Aliyun.AccessKeySecret)).
			WithRegion(config.Cfg.Aliyun.RegionId).
			WithEndpoint(config.Cfg.Aliyun.OssEndpoint()),
	)
	return client
}

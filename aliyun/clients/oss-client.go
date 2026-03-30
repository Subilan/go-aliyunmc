package clients

import (
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"go-aliyunmc/config"
)

// OssClient 是系统全局对象存储服务客户端
var OssClient *oss.Client

// GetOssClient 根据凭据创建一个对象存储服务客户端
func GetOssClient() *oss.Client {
	client := oss.NewClient(
		oss.LoadDefaultConfig().
			WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					config.G.Aliyun.AccessKeyId,
					config.G.Aliyun.AccessKeySecret,
				),
			).
			WithRegion(config.G.Aliyun.RegionId).
			WithEndpoint(config.G.Aliyun.OssEndpoint()),
	)
	return client
}

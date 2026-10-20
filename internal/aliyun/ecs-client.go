// Package aliyun 创建并暴露阿里云服务客户端（ECS、VPC、BSS），供全局调用。
package aliyun

import (
	"github.com/Subilan/go-aliyunmc/internal/config"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

// EcsClient 是系统全局云服务器服务客户端
var EcsClient *ecs20140526.Client

// ShouldCreateEcsClient 根据凭据创建一个云服务器服务客户端
func ShouldCreateEcsClient() (*ecs20140526.Client, error) {
	return ecs20140526.NewClient(&openapi.Config{
		Credential: MustGetAKCredential(),
		Endpoint:   tea.String(config.Get().Aliyun.EcsEndpoint()),
	})
}

// Package clients 提供与阿里云服务相关的客户端实例创建方法，并以全局变量的形式暴露，在整个系统中随处调用。
//
// 目前系统中主要使用到三种阿里云服务的客户端，分别对应三种阿里云的服务：
//   - 云服务器（ECS），用于对实例进行管理、查询等，是系统使用的核心阿里云服务。
//   - 专有网络（VPC），用于查询和创建实例的默认交换机，以顺利完成实例的创建。
//   - 费用（BSS），用于查询与系统运转相关的账单信息，实现统计和展示。
package clients

import (
	"github.com/Subilan/go-aliyunmc/config"
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
		Endpoint:   tea.String(config.Cfg.Aliyun.EcsEndpoint()),
	})
}

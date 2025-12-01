package clients

import (
	"github.com/Subilan/gomc-server/config"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

func ShouldCreateEcsClient() (client *ecs20140526.Client, err error) {
	return ecs20140526.NewClient(&openapi.Config{
		Credential: MustGetAKCredential(),
		Endpoint:   tea.String(config.Cfg.Aliyun.EcsEndpoint()),
	})
}

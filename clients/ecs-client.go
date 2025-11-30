package clients

import (
	"github.com/Subilan/gomc-server/config"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
)

func ShouldCreateEcsClient() (client *ecs20140526.Client, err error) {
	crConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(config.Cfg.Aliyun.AccessKeyId).
		SetAccessKeySecret(config.Cfg.Aliyun.AccessKeySecret)

	cr, err := credentials.NewCredential(crConfig)

	if err != nil {
		return nil, err
	}

	openApiConfig := &openapi.Config{
		Credential: cr,
		Endpoint:   tea.String(config.Cfg.Aliyun.Ecs.Endpoint()),
	}

	result, err := ecs20140526.NewClient(openApiConfig)

	if err != nil {
		return nil, err
	}

	return result, nil
}

package clients

import (
	"github.com/Subilan/go-aliyunmc/config"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc20160428 "github.com/alibabacloud-go/vpc-20160428/v6/client"
)

func ShouldCreateVpcClient() (*vpc20160428.Client, error) {
	return vpc20160428.NewClient(&openapi.Config{
		Credential: MustGetAKCredential(),
		Endpoint:   tea.String(config.Cfg.Aliyun.VpcEndpoint()),
	})
}

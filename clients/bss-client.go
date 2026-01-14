package clients

import (
	bss20171214 "github.com/alibabacloud-go/bssopenapi-20171214/v6/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

// BssClient 是系统全局费用服务客户端
var BssClient *bss20171214.Client

// ShouldCreateBssClient 根据凭据创建一个费用服务客户端
func ShouldCreateBssClient() (*bss20171214.Client, error) {
	return bss20171214.NewClient(&openapi.Config{
		Credential: MustGetAKCredential(),
		Endpoint:   tea.String("business.aliyuncs.com"),
	})
}

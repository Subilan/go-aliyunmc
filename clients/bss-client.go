package clients

import (
	bss20171214 "github.com/alibabacloud-go/bssopenapi-20171214/v6/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

var BssClient *bss20171214.Client

func ShouldCreateBssClient() (client *bss20171214.Client, err error) {
	return bss20171214.NewClient(&openapi.Config{
		Credential: MustGetAKCredential(),
		Endpoint:   tea.String("business.aliyuncs.com"),
	})
}

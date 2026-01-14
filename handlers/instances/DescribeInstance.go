package instances

import (
	"fmt"
	"net/http"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/helpers"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

func HandleDescribeInstance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		instanceId := c.Param("instanceId")

		if instanceId == "" {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "no instanceId provided"}
		}

		describeInstancesRequest := ecs20140526.DescribeInstancesRequest{
			RegionId:    &config.Cfg.Aliyun.RegionId,
			InstanceIds: tea.String(fmt.Sprintf("[\"%s\"]", instanceId)),
		}

		describeInstancesResponse, err := clients.EcsClient.DescribeInstances(&describeInstancesRequest)

		if err != nil {
			return nil, err
		}

		instances := describeInstancesResponse.Body.Instances.Instance

		if len(instances) == 0 {
			return helpers.Data(gin.H{}), nil
		} else {
			return helpers.Data(instances[0]), nil
		}
	})
}

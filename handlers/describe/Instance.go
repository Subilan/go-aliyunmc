package describe

import (
	"fmt"
	"net/http"

	"github.com/Subilan/gomc-server/config"
	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

func Instance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		instanceId := c.Param("instanceId")

		if instanceId == "" {
			return nil, helpers.HttpError{Code: http.StatusBadRequest, Details: "no instanceId provided"}
		}

		describeInstancesRequest := ecs20140526.DescribeInstancesRequest{
			RegionId:    &config.Cfg.Aliyun.RegionId,
			InstanceIds: tea.String(fmt.Sprintf("[\"%s\"]", instanceId)),
		}

		describeInstancesResponse, err := globals.EcsClient.DescribeInstances(&describeInstancesRequest)

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

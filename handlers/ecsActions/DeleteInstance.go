package ecsActions

import (
	"net/http"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/gin-gonic/gin"
)

func DeleteInstance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		instanceId := c.Param("instanceId")

		if instanceId == "" {
			return nil, helpers.HttpError{Code: http.StatusBadRequest, Details: "no instanceId provided"}
		}

		_, force := c.GetQuery("force")
		_, forceStop := c.GetQuery("forceStop")

		deleteInstanceRequest := &ecs20140526.DeleteInstanceRequest{
			InstanceId: &instanceId,
			Force:      &force,
			ForceStop:  &forceStop,
		}

		_, err := globals.EcsClient.DeleteInstance(deleteInstanceRequest)

		if err != nil {
			return nil, err
		}

		return gin.H{}, nil
	})
}

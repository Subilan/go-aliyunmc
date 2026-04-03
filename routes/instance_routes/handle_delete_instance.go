package instance_routes

import (
	"go-aliyunmc/aliyun"
	"go-aliyunmc/h"
	"go-aliyunmc/store"
	"net/http"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

func HandleDeleteActiveInstance(c *gin.Context) (any, error) {
	instance, err := store.GetActiveInstance()

	if err != nil {
		return nil, err
	}

	if instance == nil {
		return nil, h.HttpError(http.StatusNotFound, "无活跃实例可删除")
	}

	deleteInstanceRequest := &ecs20140526.DeleteInstanceRequest{
		InstanceId: tea.String(instance.InstanceId),
		Force:      tea.Bool(true),
		ForceStop:  tea.Bool(true),
	}

	_, err = aliyun.EcsClient.DeleteInstance(deleteInstanceRequest)
	if err != nil {
		return nil, err
	}

	err = store.DeleteActiveInstance()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

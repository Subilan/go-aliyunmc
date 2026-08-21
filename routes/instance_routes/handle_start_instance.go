package instance_routes

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/aliyun"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/Subilan/go-aliyunmc/store"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

// HandleStartInstance 启动当前活跃实例。
func HandleStartInstance(c *gin.Context) (any, error) {
	instance, err := store.GetActiveInstance()

	if err != nil {
		return nil, err
	}

	if aliyun.EcsClient == nil {
		return nil, h.HttpError(http.StatusInternalServerError, "ECS 客户端未初始化")
	}

	snap := monitors.GetInstanceStatusMonitor().Snapshot()
	if snap.IsValid() && snap.Value == "Running" {
		return nil, h.HttpError(http.StatusConflict, "实例已处于运行状态")
	}

	startInstanceRequest := &ecs20140526.StartInstanceRequest{
		InstanceId: tea.String(instance.InstanceId),
	}

	if _, err := aliyun.EcsClient.StartInstance(startInstanceRequest); err != nil {
		return nil, err
	}

	return nil, nil
}

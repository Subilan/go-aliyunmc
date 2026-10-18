package instance_routes

import (
	"errors"
	"net/http"

	"github.com/Subilan/go-aliyunmc/aliyun"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
	"github.com/Subilan/go-aliyunmc/tasks"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func HandleDeleteActiveInstance(c *gin.Context) (any, error) {
	// 如果请求中包含 ignore_non_existent=true 或 ignore_non_existent=1，则在实例不存在时不返回错误。默认为 false。
	ignoreNonExistent := c.Query("ignore_non_existent") == "true" || c.Query("ignore_non_existent") == "1"

	instance, err := store.GetActiveInstance()

	if err != nil {
		return nil, err
	}

	if err := checkRunningTasks(); err != nil {
		return nil, err
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
		if ignoreNonExistent && errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return nil, nil
}

// checkRunningTasks 在删除实例前检查是否有依赖该实例的独占任务正在执行。
func checkRunningTasks() error {
	blockers := []struct {
		key  models.TaskType
		name string
	}{
		{models.TaskTypeArchive, "归档任务"},
		{models.TaskTypeBackup, "备份任务"},
		{tasks.TaskExclusiveGroupDeployChain, "部署链路任务（创建实例 / 部署 / 启动服务器 / 启动游戏）"},
	}

	for _, blocker := range blockers {
		if tasks.IsExclusiveTaskRunning(blocker.key) {
			return h.HttpError(http.StatusConflict, blocker.name+"正在执行中，请先中止该任务后再删除实例")
		}
	}

	return nil
}

package instance_routes

import (
	"errors"
	"github.com/Subilan/go-aliyunmc/aliyun"
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

	interruptTask(models.TaskTypeArchive)
	interruptTask(models.TaskTypeBackup)

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

func interruptTask(taskType models.TaskType) {
	exec, ok := tasks.GetExclusiveExecutor(taskType)
	if !ok {
		return
	}
	exec.Interrupt()
	<-exec.Done()
}

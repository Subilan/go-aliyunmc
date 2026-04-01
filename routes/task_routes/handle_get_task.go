package task_routes

import (
	"go-aliyunmc/contextutil"
	"go-aliyunmc/h"
	"go-aliyunmc/tasks"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleGetTask(c *gin.Context) (any, error) {
	taskId, ok := contextutil.ParamUint(c, "id")

	if !ok {
		return nil, h.HttpError(http.StatusBadRequest, "无效的任务ID")
	}

	task, err := tasks.GetTask(taskId)

	if err != nil {
		return nil, err
	}

	return task, nil
}

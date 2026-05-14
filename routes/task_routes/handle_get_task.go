package task_routes

import (
	"github.com/Subilan/go-aliyunmc/context_util"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/store"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleGetTask(c *gin.Context) (any, error) {
	taskId, ok := context_util.ParamUint(c, "id")

	if !ok {
		return nil, h.HttpError(http.StatusBadRequest, "无效的任务ID")
	}

	task, err := store.GetTask(taskId)

	if err != nil {
		return nil, err
	}

	return task, nil
}

package task_routes

import (
	"errors"
	"go-aliyunmc/context_util"
	"go-aliyunmc/h"
	"go-aliyunmc/store/models"
	"go-aliyunmc/tasks"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TriggerTaskExecutionRequest struct {
	Type models.TaskType `json:"type" binding:"required"`
	Args map[string]any  `json:"args"`
}

func HandleTriggerTaskExecution(body TriggerTaskExecutionRequest, c *gin.Context) (any, error) {
	user, ok := context_util.GetUser(c)

	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	_, task, err := tasks.TriggerTask(body.Type, user, body.Args)

	if err != nil {
		if errors.Is(err, tasks.ErrTaskTypeNotFound) {
			return nil, h.HttpError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, tasks.ErrTaskTypeExecuting) {
			return nil, h.HttpError(http.StatusConflict, err.Error())
		}
		return nil, err
	}

	return task, nil
}

package task_routes

import (
	"errors"
	"go-aliyunmc/contextutil"
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
	userId, ok := contextutil.GetUserID(c)

	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	def := tasks.GetTaskDefinition(body.Type)

	if def == nil {
		return nil, h.HttpError(http.StatusNotFound, "找不到该任务类型")
	}

	executor := tasks.NewExecutor(def)

	task, err := executor.RunTask(&userId, body.Args)

	if err != nil {
		if errors.Is(err, tasks.ErrTaskTypeExecuting) {
			return nil, h.HttpError(http.StatusConflict, err.Error())
		}
		return nil, err
	}

	return task, nil
}

package task_routes

import (
	"go-aliyunmc/contextutil"
	"go-aliyunmc/h"
	"go-aliyunmc/store/models"
	"go-aliyunmc/tasks"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TriggerTaskExecutionRequest struct {
	Type models.TaskType `json:"type" binding:"required"`
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

	task, err := executor.RunTask(&userId)
	
	if err != nil {
		return nil, err
	}

	return task, nil
}
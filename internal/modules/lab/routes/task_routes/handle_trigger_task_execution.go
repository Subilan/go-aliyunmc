package task_routes

import (
	"errors"
	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/tasks"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TriggerTaskExecutionRequest struct {
	Type labmodels.TaskType `json:"type" binding:"required"`
	Args map[string]any     `json:"args"`
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

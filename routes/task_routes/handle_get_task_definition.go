package task_routes

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/tasks"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleGetTaskDefinition(c *gin.Context) (any, error) {
	taskType := c.Param("taskType")
	def, ok := tasks.TryGetTaskDefinition(taskType)

	if !ok {
		return nil, h.HttpError(http.StatusNotFound, "任务类型无效")
	}

	return def, nil
}

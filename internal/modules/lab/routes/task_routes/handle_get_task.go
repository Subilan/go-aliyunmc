package task_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleGetTask(c *gin.Context) (any, error) {
	taskId, ok := context_util.ParamUint(c, "id")

	if !ok {
		return nil, h.HttpError(http.StatusBadRequest, "无效的任务ID")
	}

	task, err := dao.GetTask(taskId)

	if err != nil {
		return nil, err
	}

	return task, nil
}

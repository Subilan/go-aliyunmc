package tasks

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/tasks"
	"github.com/gin-gonic/gin"
)

// HandleCancelTask 尝试通过上下文取消一个正在进行的任务。如果在任务上下文表中找不到该任务，该接口不会报错，而是返回一个 false
//
//	@Summary		取消任务
//	@Description	根据任务标识符，尝试取消一个可能正在运行的任务
//	@Tags			tasks, admin
//	@Param			taskId	path	string	true	"要取消的任务标识符"
//	@Produce		json
//	@Success		200	{object}	helpers.DataResp[bool]
//	@Failure		400	{object}	helpers.ErrorResp
//	@Router			/task/cancel/{taskId} [get]
func HandleCancelTask() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		taskId := c.Param("taskId")

		if taskId == "" {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "未提供taskID"}
		}

		ok := tasks.CancelById(taskId)

		return helpers.Data(ok), nil
	})
}

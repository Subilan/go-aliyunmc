package tasks

import (
	"net/http"

	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/helpers/tasks"
	"github.com/gin-gonic/gin"
)

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

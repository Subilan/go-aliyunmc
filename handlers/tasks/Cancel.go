package tasks

import (
	"net/http"

	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/remote"
	"github.com/gin-gonic/gin"
)

func Cancel() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		taskId := c.Param("taskId")

		if taskId == "" {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "未提供taskID"}
		}

		ok := remote.CancelTask(taskId)

		return helpers.Data(ok), nil
	})
}

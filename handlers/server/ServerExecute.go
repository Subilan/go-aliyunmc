package server

import (
	"fmt"
	"net/http"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type ExecuteOnServerQuery struct {
	CommandType globals.CommandType `form:"commandType" binding:"required"`
	WithOutput  bool                `form:"withOutput"`
}

// HandleServerExecute 尝试在活动实例上运行一个操作，该操作必须在预先固定的有限操作中选取一个。
func HandleServerExecute() gin.HandlerFunc {
	return helpers.QueryHandler[ExecuteOnServerQuery](func(body ExecuteOnServerQuery, c *gin.Context) (any, error) {
		userId, exists := c.Get("user_id")

		if !exists {
			return nil, &helpers.HttpError{Code: http.StatusUnauthorized, Details: "cannot get user id"}
		}

		userIdInt := userId.(int)

		activeInstance := store.GetActiveInstance()

		if activeInstance == nil || activeInstance.Ip == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "no active ip-allocated instance present"}
		}

		cmd, ok := globals.ShouldGetCommand(body.CommandType)

		if !ok {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "command not found"}
		}

		ctx, cancel := cmd.TimeoutContext()
		defer cancel()

		output, err := cmd.Run(ctx, *activeInstance.Ip, &userIdInt, &globals.CommandRunOption{Output: body.WithOutput})

		if err != nil {
			return nil, &helpers.HttpError{Code: http.StatusInternalServerError, Details: fmt.Sprintf("command failed with error %s\noutput:\n%s", err.Error(), output)}
		}

		return helpers.Data(output), nil
	})
}

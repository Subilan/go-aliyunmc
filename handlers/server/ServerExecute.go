package server

import (
	"fmt"
	"net/http"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
	"github.com/Subilan/go-aliyunmc/helpers/gctx"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type ExecuteOnServerQuery struct {
	CommandType consts.CommandType `form:"commandType" binding:"required"`
	WithOutput  bool               `form:"withOutput"`
}

// HandleServerExecute 尝试在活动实例上运行一个操作，该操作必须在预先固定的有限操作中选取一个。
func HandleServerExecute() gin.HandlerFunc {
	return helpers.QueryHandler[ExecuteOnServerQuery](func(body ExecuteOnServerQuery, c *gin.Context) (any, error) {
		userId, err := gctx.ShouldGetUserId(c)

		if err != nil {
			return nil, err
		}

		activeInstance, err := store.GetDeployedActiveInstance()

		if err != nil {
			return nil, err
		}

		cmd, ok := commands.ShouldGetCommand(body.CommandType)

		if !ok {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "command not found"}
		}

		ctx, cancel := cmd.DefaultContext()
		defer cancel()

		output, err := cmd.Run(ctx, *activeInstance.Ip, &userId, &commands.CommandRunOption{Output: body.WithOutput})

		if err != nil {
			return nil, &helpers.HttpError{Code: http.StatusInternalServerError, Details: fmt.Sprintf("command failed with error %s", err.Error())}
		}

		return helpers.Data(output), nil
	})
}

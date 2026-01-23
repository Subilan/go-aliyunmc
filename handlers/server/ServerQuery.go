package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type QueryOnServerQuery struct {
	QueryType consts.CommandType `form:"queryType" binding:"required,oneof=get_server_sizes screenfetch get_ops get_cached_players get_server_properties"`
}

func HandleServerQuery() gin.HandlerFunc {
	return helpers.QueryHandler[QueryOnServerQuery](func(query QueryOnServerQuery, c *gin.Context) (any, error) {
		activeInstance, err := store.GetDeployedActiveInstance()

		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()

		cmd, ok := commands.ShouldGetCommand(query.QueryType)

		if !ok {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "query type not found"}
		}

		if !cmd.TestRole(c) {
			return nil, &helpers.HttpError{Code: http.StatusForbidden, Details: "无权执行"}
		}

		output, err := cmd.Run(ctx, *activeInstance.Ip, nil, nil)

		if err != nil {
			return nil, err
		}

		return helpers.Data(output), nil
	})
}

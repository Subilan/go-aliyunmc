package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type QueryOnServerQuery struct {
	QueryType globals.CommandType `form:"queryType" binding:"required,oneof=sizes screenfetch"`
}

func HandleServerQuery() gin.HandlerFunc {
	return helpers.QueryHandler[QueryOnServerQuery](func(query QueryOnServerQuery, c *gin.Context) (any, error) {
		activeInstance := store.GetActiveInstance()

		if activeInstance == nil || activeInstance.Ip == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "no active ip-allocated instance present"}
		}

		ctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()

		cmd, ok := globals.ShouldGetCommand(query.QueryType)

		if !ok {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "query type not found"}
		}

		output, err := cmd.Run(ctx, *activeInstance.Ip, nil, nil)

		if err != nil {
			return nil, &helpers.HttpError{Code: http.StatusInternalServerError, Details: fmt.Sprintf("command failed with error %s\noutput:\n%s", err.Error(), output)}
		}

		return helpers.Data(output), nil
	})
}

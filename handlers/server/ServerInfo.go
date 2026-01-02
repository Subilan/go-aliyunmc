package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
	"github.com/mcstatus-io/mcutil/v4/query"
	"github.com/mcstatus-io/mcutil/v4/response"
)

var QueryFull *response.QueryFull
var QueryFullRefreshedAt time.Time

func tryGetActiveInstanceWithRunningServer() (*store.Instance, error) {
	activeInstance := store.GetActiveInstance()

	if activeInstance == nil {
		return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "no active instance"}
	}

	if !globals.IsServerRunning {
		return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "not running"}
	}

	return activeInstance, nil
}

func RefreshQueryFull(parentContext context.Context) error {
	activeInstance, err := tryGetActiveInstanceWithRunningServer()

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parentContext, time.Second*5)
	defer cancel()

	info, err := query.Full(ctx, activeInstance.Ip, 25565)

	if err != nil {
		return err
	}

	QueryFull = info
	QueryFullRefreshedAt = time.Now()

	return nil
}

func HandleRefreshServerPlayers() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		err := RefreshQueryFull(c.Request.Context())

		if err != nil {
			return nil, err
		}

		return helpers.Data(gin.H{"players": QueryFull.Players, "refreshedAt": QueryFullRefreshedAt}), nil
	})
}

func HandleGetServerPlayers() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		_, err := tryGetActiveInstanceWithRunningServer()

		if err != nil {
			return nil, err
		}

		return helpers.Data(gin.H{"players": QueryFull.Players, "refreshedAt": QueryFullRefreshedAt}), nil
	})
}

func HandleGetServerInfo() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		return helpers.Data(gin.H{"running": globals.IsServerRunning, "info": globals.ServerStatus}), nil
	})
}

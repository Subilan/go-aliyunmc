package server

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
	"github.com/mcstatus-io/mcutil/v4/response"
)

type GetServerInfoResponse struct {
	Data          *response.StatusModern `json:"data,omitempty"`
	OnlinePlayers []string               `json:"onlinePlayers,omitempty"`
	Running       bool                   `json:"running"`
}

func HandleGetServerInfo() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		activeInstance := store.GetActiveInstance()

		if activeInstance == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "no active instance found"}
		}

		if !globals.IsServerRunning {
			return helpers.Data(GetServerInfoResponse{Running: false}), nil
		}

		return helpers.Data(GetServerInfoResponse{Running: true, Data: globals.ServerStatus, OnlinePlayers: globals.OnlinePlayers}), nil
	})
}

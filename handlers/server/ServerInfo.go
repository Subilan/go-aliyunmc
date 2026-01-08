package server

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/monitors"
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
		_, err := store.GetIpAllocatedActiveInstance()

		if err != nil {
			return nil, err
		}

		if !monitors.SnapshotIsServerRunning() {
			return helpers.Data(GetServerInfoResponse{Running: false}), nil
		}

		return helpers.Data(GetServerInfoResponse{Running: true, Data: monitors.SnapshotServerStatus(), OnlinePlayers: monitors.SnapshotOnlinePlayers()}), nil
	})
}

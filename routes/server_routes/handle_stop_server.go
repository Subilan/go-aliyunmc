package server_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/server"
	"go-aliyunmc/states"
	"go-aliyunmc/store"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleStopServer(c *gin.Context) (any, error) {
	ip, err := store.GetActiveInstanceIpNonEmpty()

	if err != nil {
		return nil, err
	}

	status, ok := states.SnapshotServerStatus()
	
	if !ok || status.Error != nil {
		return nil, h.HttpError(http.StatusServiceUnavailable, "无法获取最新的服务器状态")
	}
	
	if !status.Value.Online {
		return nil, h.HttpError(http.StatusConflict, "服务器已离线")
	}

	_, err = server.RunSingleCommand(ip, "stop")
	
	if err != nil {
		return nil, err
	}

	return nil, nil
}
package server_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/monitors"
	"go-aliyunmc/server"
	"go-aliyunmc/store"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcstatus-io/mcutil/v4/query"
)

func HandleQueryServer(c *gin.Context) (any, error) {
	instance, err := store.GetActiveInstanceDefaultNil()

	if err != nil {
		return nil, err
	}

	if instance == nil {
		return nil, h.HttpError(http.StatusNotFound, "未找到实例")
	}

	snap := monitors.GetServerStatusMonitor().Snapshot()

	if !snap.IsValid() {
		return nil, h.HttpError(http.StatusServiceUnavailable, "服务器状态不可用")
	}

	if !snap.Value.Online {
		return nil, h.HttpError(http.StatusServiceUnavailable, "服务器离线")
	}

	result, err := query.Full(c.Request.Context(), instance.Ip, server.C.PortOrDefault())

	if err != nil {
		return nil, err
	}

	return result, nil
}

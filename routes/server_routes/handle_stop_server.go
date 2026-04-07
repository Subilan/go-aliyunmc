package server_routes

import (
	"go-aliyunmc/server"
	"go-aliyunmc/store"

	"github.com/gin-gonic/gin"
)

func HandleStopServer(c *gin.Context) (any, error) {
	ip, err := store.GetActiveInstanceIpNonEmpty()

	if err != nil {
		return nil, err
	}

	_, err = server.RunSingleCommand(ip, "stop")
	
	if err != nil {
		return nil, err
	}

	return nil, nil
}
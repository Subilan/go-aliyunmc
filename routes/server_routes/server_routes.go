package server_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(engine *gin.Engine) {
	serverGroup := engine.Group("/server")
	serverGroup.Use(mid.Auth())
	serverGroup.Use(mid.Perm())
	{
		serverGroup.GET("/stop", h.G(HandleStopServer))
		serverGroup.GET("/data", h.G(HandleGetData))
		serverGroup.GET("/query", h.G(HandleQueryServer))
	}
}

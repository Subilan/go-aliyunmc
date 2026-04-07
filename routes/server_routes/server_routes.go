package server_routes

import (
	"go-aliyunmc/h"

	"github.com/gin-gonic/gin"
)

func Bind(engine *gin.Engine) {
	serverGroup := engine.Group("/server")
	{
		serverGroup.GET("/stop", h.G(HandleStopServer))
	}
}

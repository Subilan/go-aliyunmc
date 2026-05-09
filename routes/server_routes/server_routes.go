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
		serverGroup.GET("/leaderboard", h.Q(HandleLeaderboard))
		serverGroup.GET("/ip", h.G(HandleGetIP))

		gameGroup := serverGroup.Group("")
		gameGroup.Use(mid.Whitelisted())
		{
			gameGroup.GET("/stats/:uuid", h.G(HandleGetStats))
			gameGroup.GET("/advancements/:uuid", h.G(HandleGetAdvancements))
			gameGroup.GET("/player-list", h.G(HandleGetPlayerList))
		}
	}
}

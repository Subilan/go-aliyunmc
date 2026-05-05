package sample_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	sampleGroup := router.Group("/samples")

	authorized := sampleGroup.Group("")
	authorized.Use(mid.Auth())
	authorized.Use(mid.Perm())

	{
		authorized.GET("/player-list-history", h.G(HandleGetPlayerListHistory))
		authorized.GET("/account-balance-history", h.G(HandleGetAccountBalanceHistory))
	}
}

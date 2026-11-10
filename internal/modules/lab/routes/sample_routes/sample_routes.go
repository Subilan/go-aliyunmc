package sample_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/mid"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.RouterGroup) {
	sampleGroup := router.Group("/samples")

	authorized := sampleGroup.Group("")
	authorized.Use(mid.Auth())
	authorized.Use(mid.Perm())

	{
		authorized.GET("/player-list-history", h.G(HandleGetPlayerListHistory))
		authorized.GET("/player-online-ranges", h.G(HandleGetPlayerOnlineRanges))
		authorized.GET("/account-balance-history", h.G(HandleGetAccountBalanceHistory))
		authorized.GET("/account-balance-stats", h.G(HandleGetAccountBalanceStats))
	}
}

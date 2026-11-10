package bss_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/mid"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.RouterGroup) {
	bssGroup := router.Group("/bss")
	authorized := bssGroup.Use(mid.Auth())
	{
		authorized.GET("/balance", h.G(HandleGetBalance))
	}
}

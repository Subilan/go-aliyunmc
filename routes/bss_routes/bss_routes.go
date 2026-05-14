package bss_routes

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	bssGroup := router.Group("/bss")
	authorized := bssGroup.Use(mid.Auth())
	{
		authorized.GET("/balance", h.G(HandleGetBalance))
	}
}
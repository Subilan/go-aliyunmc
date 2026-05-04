package bss_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	bssGroup := router.Group("/bss")
	authorized := bssGroup.Use(mid.Auth())
	{
		authorized.GET("/balance", h.G(HandleGetBalance))
	}
}
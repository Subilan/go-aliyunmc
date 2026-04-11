package monitor_routes

import (
	"go-aliyunmc/global_states"
	"go-aliyunmc/h"
	"go-aliyunmc/mid"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	monitorGroup := router.Group("/monitor")
	authorized := monitorGroup.Group("")
	authorized.Use(mid.Auth())
	aai := authorized.Group("/auto-archive-idle")
	{
		aai.GET("/remaining-secs", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, h.Data(global_states.GetApproxIdleRemaningSecs()))
		})
	}
}
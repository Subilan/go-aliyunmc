package monitor_routes

import (
	"go-aliyunmc/global_states"
	"go-aliyunmc/h"
	"go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	monitorGroup := router.Group("/monitor")
	authorized := monitorGroup.Group("")
	authorized.Use(mid.Auth())
	authorized.Use(mid.Perm())
	aai := authorized.Group("/auto-archive-idle")
	{
		aai.GET("/remaining-secs", h.V(global_states.GetApproxIdleRemaningSecs))
	}
}
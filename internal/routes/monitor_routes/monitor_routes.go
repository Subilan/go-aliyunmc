package monitor_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/global_states"
	"github.com/Subilan/go-aliyunmc/internal/h"
	"github.com/Subilan/go-aliyunmc/internal/mid"

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

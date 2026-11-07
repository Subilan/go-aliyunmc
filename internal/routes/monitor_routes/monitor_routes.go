package monitor_routes

import (
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
		aai.GET("", h.G(HandleGetIdleCountdown))
		aai.POST("", h.B(HandleSetIdleCountdown))
		aai.POST("/pause", h.G(HandlePauseIdleCountdown))
		aai.POST("/resume", h.G(HandleResumeIdleCountdown))
		aai.POST("/reset", h.G(HandleResetIdleCountdown))
		aai.GET("/remaining-secs", h.V(HandleGetIdleCountdownRemainingSecs))
	}
}

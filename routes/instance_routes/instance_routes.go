package instance_routes

import (
	"github.com/Subilan/go-aliyunmc/global_states"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/mid"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/Subilan/go-aliyunmc/states"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	instanceGroup := router.Group("/instance")
	authorized := instanceGroup.Group("")
	authorized.Use(mid.Auth())
	authorized.Use(mid.Perm())
	{
		authorized.DELETE("/active", h.G(HandleDeleteActiveInstance))
		authorized.GET("/active", h.G(HandleGetActiveInstance))
		authorized.POST("/start", h.G(HandleStartInstance))
		authorized.DELETE("/spot-interruption", h.G(HandleClearSpotInterruption))
		authorized.GET("/candidates", h.V(global_states.GetCurrentEcsCandidates))
		authorized.GET("/best-candidate", h.VB(func() (states.EcsCandidate, bool) {
			snap := monitors.GetBestEcsCandidateMonitor().Snapshot()
			return snap.Value, snap.IsValid()
		}))
	}
}

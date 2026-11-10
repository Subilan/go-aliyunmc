package instance_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/states"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/global_states"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/monitors"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.RouterGroup) {
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

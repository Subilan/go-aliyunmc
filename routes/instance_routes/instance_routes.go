package instance_routes

import (
	"go-aliyunmc/global_states"
	"go-aliyunmc/h"
	"go-aliyunmc/mid"
	"go-aliyunmc/states"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	instanceGroup := router.Group("/instance")
	authorized := instanceGroup.Group("")
	authorized.Use(mid.Auth())
	{
		authorized.DELETE("/active", h.G(HandleDeleteActiveInstance))
		authorized.GET("/candidates", h.V(global_states.GetCurrentEcsCandidates))
		authorized.GET("/best-candidate", h.VB(states.SnapshotBestEcsCandidate))
	}
}

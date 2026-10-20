package changelog_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/h"
	"github.com/Subilan/go-aliyunmc/internal/mid"

	"github.com/gin-gonic/gin"
)

func Bind(engine *gin.Engine) {
	changelogGroup := engine.Group("")
	changelogGroup.Use(mid.Auth())
	changelogGroup.Use(mid.Perm())
	{
		changelogGroup.POST("/changelog", h.B(HandleCreate))
		changelogGroup.GET("/changelogs", h.Q(HandleQuery))
		changelogGroup.POST("/changelog/:id/like", h.G(HandleLike))
		changelogGroup.DELETE("/changelog/:id", h.G(HandleDelete))
		changelogGroup.PATCH("/changelog/:id", h.B(HandleUpdate))
	}
}

package simple_routes

import (
	"runtime"

	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/internal/version"
	"github.com/Subilan/go-aliyunmc/mc"
	"github.com/Subilan/go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(engine *gin.Engine) {
	simpleGroup := engine.Group("/simple")

	authorized := simpleGroup.Use(mid.Auth())

	{
		authorized.GET("/info", h.V(func() map[string]any {
			return map[string]any{
				"version":    version.Version,
				"go_version": runtime.Version(),
				"build_time": version.BuildTime,
				"platform":   runtime.GOOS + "/" + runtime.GOARCH,
				"commit":     version.Commit,
			}
		}))

		authorized.GET("/mc-translations", h.V(func() map[string]any {
			return map[string]any{
				"biomes":         mc.Biomes(),
				"blocksAndItems": mc.BlocksAndItems(),
				"entities":       mc.Entities(),
				"stats":          mc.Stats(),
			}
		}))
	}
}

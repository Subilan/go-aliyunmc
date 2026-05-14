package simple_routes

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/mc"
	"github.com/Subilan/go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(engine *gin.Engine) {
	simpleGroup := engine.Group("/simple")

	authorized := simpleGroup.Use(mid.Auth())

	{
		authorized.GET("/mc-translations", h.V(func() map[string]any {
			return map[string]any{
				"biomes":         mc.Biomes,
				"blocksAndItems": mc.BlocksAndItems,
				"entities":       mc.Entities,
				"stats":          mc.Stats,
			}
		}))
	}
}

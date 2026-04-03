package instance_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	instanceGroup := router.Group("/instance")
	authorized := instanceGroup.Group("")
	authorized.Use(mid.Auth())
	{
		authorized.DELETE("/active", h.G(HandleDeleteActiveInstance))
	}
}

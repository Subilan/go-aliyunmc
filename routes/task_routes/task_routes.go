package task_routes

import (
	"go-aliyunmc/h"
	"go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	taskGroup := router.Group("/task")

	authorized := taskGroup.Group("")
	authorized.Use(mid.Auth())

	{
		authorized.GET("/:id/output", HandleGetTaskOutput)
		authorized.POST("/trigger", h.B(HandleTriggerTaskExecution))
	}
}

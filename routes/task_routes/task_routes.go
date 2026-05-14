package task_routes

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/mid"

	"github.com/gin-gonic/gin"
)

func Bind(router *gin.Engine) {
	taskGroup := router.Group("/task")

	authorized := taskGroup.Group("")
	authorized.Use(mid.Auth())
	authorized.Use(mid.Perm())
	{
		authorized.GET("/stats", h.G(HandleGetTaskStats))
		authorized.GET("/s", h.Q(HandleListTasks))
		authorized.GET("/:id", h.G(HandleGetTask))
		authorized.GET("/definition/:taskType", h.G(HandleGetTaskDefinition))
		authorized.GET("/:id/output", HandleGetTaskOutput)
		authorized.POST("/trigger", mid.Whitelisted(), h.B(HandleTriggerTaskExecution))
	}
}

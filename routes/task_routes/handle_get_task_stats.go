package task_routes

import (
	"go-aliyunmc/store"

	"github.com/gin-gonic/gin"
)

func HandleGetTaskStats(c *gin.Context) (any, error) {
	return store.GetTaskStats()
}

package task_routes

import (
	"github.com/Subilan/go-aliyunmc/store"

	"github.com/gin-gonic/gin"
)

func HandleGetTaskStats(c *gin.Context) (any, error) {
	return store.GetTaskStats()
}

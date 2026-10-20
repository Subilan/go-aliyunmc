package task_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/store"

	"github.com/gin-gonic/gin"
)

func HandleGetTaskStats(c *gin.Context) (any, error) {
	return store.GetTaskStats()
}

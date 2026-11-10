package task_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"

	"github.com/gin-gonic/gin"
)

func HandleGetTaskStats(c *gin.Context) (any, error) {
	return dao.GetTaskStats()
}

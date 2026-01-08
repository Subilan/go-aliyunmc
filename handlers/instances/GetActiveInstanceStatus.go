package instances

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/gin-gonic/gin"
)

func HandleGetActiveInstanceStatus() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		instanceStatus := monitors.SnapshotInstanceStatus()

		return helpers.Data(instanceStatus), nil
	})
}

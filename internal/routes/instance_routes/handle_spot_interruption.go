package instance_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/global_states"
	"github.com/gin-gonic/gin"
)

// HandleClearSpotInterruption 手动解除抢占式实例回收保护状态。
func HandleClearSpotInterruption(c *gin.Context) (any, error) {
	global_states.ClearSpotInterruption()
	return nil, nil
}

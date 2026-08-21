package instance_routes

import (
	"github.com/Subilan/go-aliyunmc/global_states"
	"github.com/gin-gonic/gin"
)

// HandleClearSpotInterruption 手动解除抢占式实例回收保护状态。
//
// 正常情况下保护状态会在实例恢复 Running 后自动解除，本接口仅在自动解除失败时作为
// 管理员兜底手段使用。
func HandleClearSpotInterruption(c *gin.Context) (any, error) {
	global_states.ClearSpotInterruption()
	return nil, nil
}

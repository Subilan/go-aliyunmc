package sample_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/monitors"

	"github.com/gin-gonic/gin"
)

func HandleGetPlayerListHistory(c *gin.Context) (any, error) {
	sampler := monitors.GetPlayerListSampler()
	if sampler == nil {
		return []monitors.PlayerListDataPoint{}, nil
	}
	return sampler.Snapshot(), nil
}

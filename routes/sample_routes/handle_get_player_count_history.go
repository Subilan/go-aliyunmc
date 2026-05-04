package sample_routes

import (
	"go-aliyunmc/monitors"

	"github.com/gin-gonic/gin"
)

func HandleGetPlayerListHistory(c *gin.Context) (any, error) {
	sampler := monitors.GetPlayerListSampler()
	if sampler == nil {
		return []monitors.PlayerListDataPoint{}, nil
	}
	return sampler.Snapshot(), nil
}

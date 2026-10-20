package sample_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/monitors"

	"github.com/gin-gonic/gin"
)

func HandleGetAccountBalanceStats(c *gin.Context) (any, error) {
	sampler := monitors.GetBalanceSampler()
	if sampler == nil {
		return monitors.BalanceStats{
			MinRunningHours: monitors.BalanceStatsMinSpan.Hours(),
		}, nil
	}
	return sampler.Stats(), nil
}

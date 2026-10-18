package sample_routes

import (
	"github.com/Subilan/go-aliyunmc/monitors"

	"github.com/gin-gonic/gin"
)

func HandleGetAccountBalanceStats(c *gin.Context) (any, error) {
	sampler := monitors.GetBalanceSampler()
	if sampler == nil {
		return monitors.BalanceStats{
			MinSpanHours: monitors.BalanceStatsMinSpan.Hours(),
		}, nil
	}
	return sampler.Stats(), nil
}

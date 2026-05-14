package sample_routes

import (
	"github.com/Subilan/go-aliyunmc/monitors"

	"github.com/gin-gonic/gin"
)

func HandleGetAccountBalanceHistory(c *gin.Context) (any, error) {
	sampler := monitors.GetBalanceSampler()
	if sampler == nil {
		return []monitors.BalanceDataPoint{}, nil
	}
	return sampler.Snapshot(), nil
}
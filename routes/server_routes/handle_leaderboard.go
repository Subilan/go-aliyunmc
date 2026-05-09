package server_routes

import (
	"go-aliyunmc/leaderboard"

	"github.com/gin-gonic/gin"
)

// LeaderboardQuery represents query parameters for the leaderboard endpoint
type LeaderboardQuery struct {
	Metric string `form:"metric" binding:"required"`
	Order  string `form:"order"`
}

// HandleLeaderboard returns leaderboard rankings for the requested metric
func HandleLeaderboard(q LeaderboardQuery, c *gin.Context) (any, error) {
	return leaderboard.BuildLeaderboard(leaderboard.Query{
		Metric: q.Metric,
		Order:  q.Order,
	})
}

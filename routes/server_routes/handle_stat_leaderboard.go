package server_routes

import (
	"github.com/Subilan/go-aliyunmc/leaderboard"

	"github.com/gin-gonic/gin"
)

// StatLeaderboardQuery represents query parameters for per-stat leaderboard
type StatLeaderboardQuery struct {
	Metric string `form:"metric" binding:"required"`
	Order  string `form:"order"`
}

// HandleStatLeaderboard returns rankings for a single raw Minecraft stat key
func HandleStatLeaderboard(q StatLeaderboardQuery, c *gin.Context) (any, error) {
	return leaderboard.BuildRawLeaderboard(q.Metric, q.Order)
}

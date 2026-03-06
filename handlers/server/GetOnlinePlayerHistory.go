package server

import (
	"encoding/json"
	"time"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/gin-gonic/gin"
)

type GetOnlinePlayerHistoryRequest struct {
	TimeRange string `form:"timeRange" binding:"required,oneof=1w 1d 1h 6h"`
}

type OnlinePlayerHistoryRecord struct {
	CreatedAt   time.Time `json:"createdAt"`
	PlayerCount int       `json:"playerCount"`
	Players     []string  `json:"players"`
}

func HandleGetOnlinePlayerHistory() gin.HandlerFunc {
	return helpers.QueryHandler(func(query GetOnlinePlayerHistoryRequest, c *gin.Context) (any, error) {
		var minCreatedAt time.Time

		if query.TimeRange == "1w" {
			minCreatedAt = time.Now().Add(-7 * 24 * time.Hour)
		}

		if query.TimeRange == "1d" {
			minCreatedAt = time.Now().Add(-24 * time.Hour)
		}

		if query.TimeRange == "1h" {
			minCreatedAt = time.Now().Add(-time.Hour)
		}

		if query.TimeRange == "6h" {
			minCreatedAt = time.Now().Add(-6 * time.Hour)
		}

		records, err := db.Pool.Query("SELECT created_at, `player_count`, `players` FROM `online_player_history` WHERE `created_at` > ? ORDER BY `created_at`", minCreatedAt)
		if err != nil {
			return nil, err
		}
		defer records.Close()

		var results = make([]OnlinePlayerHistoryRecord, 0)

		for records.Next() {
			var record OnlinePlayerHistoryRecord
			var playersRaw string
			err := records.Scan(&record.CreatedAt, &record.PlayerCount, &playersRaw)

			if err != nil {
				return nil, err
			}

			err = json.Unmarshal([]byte(playersRaw), &record.Players)

			if err != nil {
				return nil, err
			}

			results = append(results, record)
		}

		return helpers.Data(results), nil
	})
}

package server

import (
	"encoding/json"
	"time"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/gin-gonic/gin"
)

type GetOnlinePlayerOverviewQuery struct {
	TimeRange string `form:"timeRange" binding:"required,oneof=1w"`
}

type GetOnlinePlayerOverviewResponse struct {
	UniquePlayers        []string `json:"uniquePlayers"`
	TotalHours           float64  `json:"totalHours"`
	MaxConcurrentPlayers int      `json:"maxConcurrentPlayers"`
	ServerOnlineHours    float64  `json:"serverOnlineHours"`
}

func HandleGetOnlinePlayerOverview() gin.HandlerFunc {
	return helpers.QueryHandler(func(query GetOnlinePlayerOverviewQuery, c *gin.Context) (any, error) {
		var result GetOnlinePlayerOverviewResponse

		minCreatedAt := time.Now().Add(-7 * 24 * time.Hour)

		records, err := db.Pool.Query("SELECT `created_at`, `player_count`, `players` FROM `online_player_history` WHERE `created_at` > ? ORDER BY `created_at`", minCreatedAt)
		if err != nil {
			return nil, err
		}
		defer records.Close()

		uniquePlayers := make(map[string]bool)
		var totalPlayerHours float64
		maxConcurrentPlayers := 0
		totalRecords := 0
		onlineRecords := 0

		for records.Next() {
			var createdAt time.Time
			var playerCount int
			var playersRaw string
			err := records.Scan(&createdAt, &playerCount, &playersRaw)
			if err != nil {
				return nil, err
			}

			var players []string
			err = json.Unmarshal([]byte(playersRaw), &players)
			if err != nil {
				return nil, err
			}

			for _, player := range players {
				uniquePlayers[player] = true
			}

			if playerCount > maxConcurrentPlayers {
				maxConcurrentPlayers = playerCount
			}

			if playerCount > 0 {
				onlineRecords++
			}

			totalPlayerHours += float64(playerCount) * (5.0 / 3600.0)

			totalRecords++
		}

		result.UniquePlayers = make([]string, 0, len(uniquePlayers))
		for player := range uniquePlayers {
			result.UniquePlayers = append(result.UniquePlayers, player)
		}
		result.TotalHours = totalPlayerHours
		result.MaxConcurrentPlayers = maxConcurrentPlayers

		serverOnlineHours, err := calculateServerOnlineHours(minCreatedAt)
		if err != nil {
			return nil, err
		}
		result.ServerOnlineHours = serverOnlineHours

		return helpers.Data(result), nil
	})
}

func calculateServerOnlineHours(minCreatedAt time.Time) (float64, error) {
	records, err := db.Pool.Query(
		"SELECT `type`, `created_at` FROM `command_exec` WHERE `created_at` > ? AND `type` IN (?, ?) ORDER BY `created_at`",
		minCreatedAt,
		consts.CmdTypeStartServer,
		consts.CmdTypeStopServer,
	)
	if err != nil {
		return 0, err
	}
	defer records.Close()

	type execRecord struct {
		typ       string
		createdAt time.Time
	}

	var allRecords []execRecord
	for records.Next() {
		var typ string
		var createdAt time.Time
		err := records.Scan(&typ, &createdAt)
		if err != nil {
			return 0, err
		}
		allRecords = append(allRecords, execRecord{typ: typ, createdAt: createdAt})
	}

	if len(allRecords) == 0 {
		return 0, nil
	}

	// 如果第一条记录是 stop_server，则忽略它
	if allRecords[0].typ == string(consts.CmdTypeStopServer) {
		allRecords = allRecords[1:]
	}

	// 如果最后一条记录是 start_server，则忽略它
	if len(allRecords) > 0 && allRecords[len(allRecords)-1].typ == string(consts.CmdTypeStartServer) {
		allRecords = allRecords[:len(allRecords)-1]
	}

	if len(allRecords) == 0 {
		return 0, nil
	}

	var totalServerHours float64
	var serverStartTime time.Time
	hasActiveServer := false

	for _, record := range allRecords {
		if record.typ == string(consts.CmdTypeStartServer) {
			serverStartTime = record.createdAt
			hasActiveServer = true
		} else if record.typ == string(consts.CmdTypeStopServer) && hasActiveServer {
			serverUptime := record.createdAt.Sub(serverStartTime).Seconds()
			totalServerHours += serverUptime / 3600.0
			hasActiveServer = false
		}
	}

	return totalServerHours, nil
}

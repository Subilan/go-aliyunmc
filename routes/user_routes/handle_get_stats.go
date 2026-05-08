package user_routes

import (
	"encoding/json"
	"net/http"

	"go-aliyunmc/context_util"
	"go-aliyunmc/h"
	"go-aliyunmc/playerdata"

	"github.com/gin-gonic/gin"
)

type GameStatsResponse struct {
	Stats               json.RawMessage                 `json:"stats"`
	Playtime            float64                         `json:"playtime"`
	AdvancementProgress playerdata.AdvancementProgress  `json:"advancement_progress"`
	PlayerName          string                          `json:"player_name"`
	OnlineDates         []string                        `json:"online_dates"`
	JoinStreak          int                             `json:"join_streak"`
	LastSeen            *int64                          `json:"last_seen"`
}

func HandleGetGameStats(c *gin.Context) (any, error) {
	user, exists := context_util.GetUser(c)
	if !exists {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	uuid := *user.WhitelistUUID

	stats, err := playerdata.ReadStats(uuid)
	if err != nil {
		return nil, err
	}

	advProgress := playerdata.ReadAdvancementProgress(uuid)

	playtime, _ := playerdata.ReadMinecraftPlaytime(uuid)
	pluginPlaytime := playerdata.QueryPlaytime(uuid)

	playerName := playerdata.LookupPlayerName(uuid)

	var joinStreak int
	var lastSeen *int64
	if pluginPlaytime != nil {
		joinStreak = pluginPlaytime.JoinStreak
		lastSeen = pluginPlaytime.LastSeen
	}

	return &GameStatsResponse{
		Stats:               stats,
		Playtime:            playtime,
		AdvancementProgress: advProgress,
		PlayerName:          playerName,
		OnlineDates:         playerdata.QueryOnlineDates(playerName),
		JoinStreak:          joinStreak,
		LastSeen:            lastSeen,
	}, nil
}

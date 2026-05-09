package server_routes

import (
	"encoding/json"
	"net/http"
	"os"

	"go-aliyunmc/context_util"
	"go-aliyunmc/h"
	"go-aliyunmc/playerdata"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
)

type GameStatsResponse struct {
	Stats               json.RawMessage                `json:"stats"`
	Playtime            float64                        `json:"playtime"`
	AdvancementProgress playerdata.AdvancementProgress `json:"advancement_progress"`
	PlayerName          string                         `json:"player_name"`
	OnlineDates         []string                       `json:"online_dates"`
	JoinStreak          int                            `json:"join_streak"`
	LastSeen            *int64                         `json:"last_seen"`
}

func HandleGetStats(c *gin.Context) (any, error) {
	uuid := c.Param("uuid")
	if uuid == "" {
		return nil, h.HttpError(http.StatusBadRequest, "缺少玩家UUID参数")
	}

	// Check privacy: if the UUID is bound to a user who disallows public access,
	// only that user can view the stats.
	var owner models.User
	if err := store.DB.Where("whitelist_uuid = ?", uuid).First(&owner).Error; err == nil {
		currentUser, _ := context_util.GetUser(c)
		if currentUser != nil && currentUser.ID != owner.ID {
			prefs, err := store.GetUserPreferences(owner.ID)
			if err == nil && prefs.DisallowPublicGameStats {
				return nil, h.HttpError(http.StatusForbidden, "该玩家不允许其他人查看游戏统计")
			}
		}
	}

	stats, err := playerdata.ReadStats(uuid)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, h.HttpError(http.StatusNotFound, "未找到该玩家的统计数据")
		}
		return nil, err
	}

	advProgress := playerdata.ReadAdvancementProgress(uuid)

	playtime, _ := playerdata.ReadMinecraftPlaytime(uuid)
	pluginPlaytime := playerdata.QueryPlaytime(uuid)

	var joinStreak int
	var lastSeen *int64
	if pluginPlaytime != nil {
		joinStreak = pluginPlaytime.JoinStreak
		lastSeen = pluginPlaytime.LastSeen
	}

	playerName := playerdata.LookupPlayerName(uuid)

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

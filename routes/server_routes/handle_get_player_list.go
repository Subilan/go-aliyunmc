package server_routes

import (
	"github.com/Subilan/go-aliyunmc/playerdata"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
)

type PlayerListEntry struct {
	UUID                    string `json:"uuid"`
	Name                    string `json:"name"`
	DisallowPublicGameStats bool   `json:"disallow_public_game_stats"`
	HasData                 bool   `json:"has_data"`
}

func HandleGetPlayerList(c *gin.Context) (any, error) {
	whitelist, err := playerdata.ReadWhitelist()
	if err != nil {
		return nil, err
	}

	uuids := make([]string, len(whitelist))
	for i, entry := range whitelist {
		uuids[i] = entry.UUID
	}

	var users []models.User
	if len(uuids) > 0 {
		store.DB.Where("whitelist_uuid IN ?", uuids).Find(&users)
	}

	disallowMap := make(map[string]bool, len(users))
	for _, user := range users {
		prefs, err := store.GetUserPreferences(user.ID)
		disallowMap[*user.WhitelistUUID] = err == nil && prefs.DisallowPublicGameStats
	}

	statsUUIDs, err := playerdata.ListPlayerUUIDs()
	hasDataSet := make(map[string]bool, len(statsUUIDs))
	if err == nil {
		for _, uid := range statsUUIDs {
			hasDataSet[uid] = true
		}
	}

	result := make([]PlayerListEntry, 0, len(whitelist))
	for _, entry := range whitelist {
		result = append(result, PlayerListEntry{
			UUID:                    entry.UUID,
			Name:                    entry.Name,
			DisallowPublicGameStats: disallowMap[entry.UUID],
			HasData:                 hasDataSet[entry.UUID],
		})
	}

	return result, nil
}

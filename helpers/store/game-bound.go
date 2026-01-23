package store

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/helpers/db"
)

type GameBound struct {
	UserId      int64     `json:"userId"`
	GameId      string    `json:"gameId"`
	Whitelisted bool      `json:"whitelisted"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type WhitelistItem struct {
	Uuid string `json:"uuid"`
	Name string `json:"name"`
}

func IsWhitelisted(gameId string) bool {
	var whitelist []WhitelistItem

	whitelistContent, err := os.ReadFile(config.Cfg.Monitor.Whitelist.CacheFile)

	if err != nil {
		return false
	}

	err = json.Unmarshal(whitelistContent, &whitelist)

	if err != nil {
		return false
	}

	for _, item := range whitelist {
		if item.Name == gameId {
			return true
		}
	}

	return false
}

func GetGameBound(userId int64) (*GameBound, bool) {
	var result GameBound

	err := db.Pool.QueryRow("SELECT user_id, game_id, created_at, updated_at FROM game_bounds WHERE user_id = ?", userId).
		Scan(&result.UserId, &result.GameId, &result.CreatedAt, &result.UpdatedAt)

	result.Whitelisted = IsWhitelisted(result.GameId)

	if err != nil {
		return nil, false
	}

	return &result, true
}

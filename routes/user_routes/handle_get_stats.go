package user_routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-aliyunmc/context_util"
	"go-aliyunmc/h"
	"go-aliyunmc/mc"
	"net/http"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
)

const (
	playerStatsDir        = "remote_data_cache/player_stats"
	playerAdvancementsDir = "remote_data_cache/player_advancements"
	playtimeDB            = "remote_data_cache/playtime.db"
)

type PlaytimeInfo struct {
	Playtime           int64  `json:"playtime"`
	ArtificialPlaytime int64  `json:"artificial_playtime"`
	AfkPlaytime        int64  `json:"afk_playtime"`
	LastSeen           *int64 `json:"last_seen"`
	FirstJoin          *int64 `json:"first_join"`
	JoinStreak         int    `json:"join_streak"`
}

type CategoryProgress struct {
	Category  string `json:"category"`
	Completed int    `json:"completed"`
	Total     int    `json:"total"`
}

type AdvancementProgress struct {
	Total      int                `json:"total"`
	Completed  int                `json:"completed"`
	Categories []CategoryProgress `json:"categories"`
}

type GameStatsResponse struct {
	Stats               json.RawMessage     `json:"stats"`
	Playtime            *PlaytimeInfo       `json:"playtime"`
	AdvancementProgress AdvancementProgress `json:"advancement_progress"`
	PlayerName          string              `json:"player_name"`
}

func HandleGetGameStats(c *gin.Context) (any, error) {
	user, exists := context_util.GetUser(c)
	if !exists {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	uuid := *user.WhitelistUUID

	// 读取玩家统计数据
	statsData, err := os.ReadFile(fmt.Sprintf("%s/%s.json", playerStatsDir, uuid))
	if err != nil {
		return nil, fmt.Errorf("读取玩家统计数据失败: %w", err)
	}

	var statsWrapper struct {
		Stats       json.RawMessage `json:"stats"`
		DataVersion int             `json:"DataVersion"`
	}
	if err := json.Unmarshal(statsData, &statsWrapper); err != nil {
		return nil, fmt.Errorf("解析玩家统计数据失败: %w", err)
	}

	// 读取玩家成就数据，计算进度
	advProgress := buildAdvancementProgress(uuid)

	// 读取游玩时长
	playtime := queryPlaytime(uuid)

	// 从白名单查找玩家名称
	playerName := lookupPlayerName(uuid)

	return &GameStatsResponse{
		Stats:               statsWrapper.Stats,
		Playtime:            playtime,
		AdvancementProgress: advProgress,
		PlayerName:          playerName,
	}, nil
}

func buildAdvancementProgress(uuid string) AdvancementProgress {
	type rawAchievement struct {
		Done bool `json:"done"`
	}

	data, err := os.ReadFile(fmt.Sprintf("%s/%s.json", playerAdvancementsDir, uuid))
	if err != nil {
		return AdvancementProgress{Total: len(mc.Advancements)}
	}

	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return AdvancementProgress{Total: len(mc.Advancements)}
	}

	completed := 0
	categoryCompleted := make(map[string]int)
	categoryTotal := make(map[string]int)

	for key := range rawData {
		var raw rawAchievement
		if err := json.Unmarshal(rawData[key], &raw); err != nil {
			continue
		}
		if !raw.Done {
			continue
		}
		if _, known := mc.Advancements[key]; !known {
			continue
		}
		completed++
		category := advancementCategory(key)
		categoryCompleted[category]++
	}

	for key := range mc.Advancements {
		category := advancementCategory(key)
		categoryTotal[category]++
	}

	// build ordered category list
	categories := make([]CategoryProgress, 0, len(categoryTotal))
	for _, cat := range []string{"story", "husbandry", "adventure", "nether", "end"} {
		total, ok := categoryTotal[cat]
		if !ok {
			continue
		}
		categories = append(categories, CategoryProgress{
			Category:  "minecraft:" + cat,
			Completed: categoryCompleted[cat],
			Total:     total,
		})
	}

	return AdvancementProgress{
		Total:      len(mc.Advancements),
		Completed:  completed,
		Categories: categories,
	}
}

func advancementCategory(key string) string {
	// key format: "minecraft:story/mine_stone"
	after, ok := strings.CutPrefix(key, "minecraft:")
	if !ok {
		return ""
	}
	before, _, found := strings.Cut(after, "/")
	if !found {
		return after
	}
	return before
}

func lookupPlayerName(uuid string) string {
	entries, err := readWhitelist()
	if err != nil {
		return ""
	}
	for i := range entries {
		if entries[i].UUID == uuid {
			return entries[i].Name
		}
	}
	return ""
}

func queryPlaytime(uuid string) *PlaytimeInfo {
	db, err := sql.Open("sqlite3", playtimeDB)
	if err != nil {
		return nil
	}
	defer db.Close()

	row := db.QueryRow(
		"SELECT playtime, artificial_playtime, afk_playtime, last_seen, first_join, relative_join_streak FROM play_time WHERE uuid = ?",
		uuid,
	)

	var info PlaytimeInfo
	err = row.Scan(&info.Playtime, &info.ArtificialPlaytime, &info.AfkPlaytime,
		&info.LastSeen, &info.FirstJoin, &info.JoinStreak)
	if err != nil {
		return nil
	}

	return &info
}

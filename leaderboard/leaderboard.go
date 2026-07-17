package leaderboard

import (
	"encoding/json"
	"fmt"
	"github.com/Subilan/go-aliyunmc/playerdata"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"
	"sort"
	"strings"
)

type MetricType string

const (
	MetricMinecraftPlaytime MetricType = "minecraft_playtime"
	MetricAchievements      MetricType = "achievements"
	MetricDistance          MetricType = "distance"
	MetricMobKills          MetricType = "mob_kills"
	MetricBlocksMined       MetricType = "blocks_mined"
	MetricAvgMoveDistance   MetricType = "avg_move_distance"
	MetricLoginDays         MetricType = "login_days"
	MetricJoinStreak        MetricType = "join_streak"
)

var validMetrics = map[string]bool{
	"minecraft_playtime": true,
	"achievements":       true,
	"distance":           true,
	"mob_kills":          true,
	"blocks_mined":       true,
	"avg_move_distance":  true,
	"login_days":         true,
	"join_streak":        true,
}

// Entry represents a single player's ranking entry
type Entry struct {
	UUID       string  `json:"uuid"`
	PlayerName string  `json:"player_name"`
	Value      float64 `json:"value"`
}

// Query represents the leaderboard query parameters
type Query struct {
	Metric string
	Order  string
}

// BuildLeaderboard computes the leaderboard for the given query
func BuildLeaderboard(q Query) ([]Entry, error) {
	if !validMetrics[q.Metric] {
		return nil, fmt.Errorf("无效的指标: %s", q.Metric)
	}

	uuids, err := playerdata.ListPlayerUUIDs()
	if err != nil {
		return nil, fmt.Errorf("列出玩家失败: %w", err)
	}

	excludeUUIDs, err := buildOptOutSet()
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, uuid := range uuids {
		if excludeUUIDs[uuid] {
			continue
		}

		value, ok := computeMetric(uuid, q.Metric)
		if !ok {
			continue
		}

		entries = append(entries, Entry{
			UUID:       uuid,
			PlayerName: playerdata.LookupPlayerName(uuid),
			Value:      value,
		})
	}

	desc := q.Order != "asc"
	sort.Slice(entries, func(i, j int) bool {
		if desc {
			return entries[i].Value > entries[j].Value
		}
		return entries[i].Value < entries[j].Value
	})

	if entries == nil {
		entries = []Entry{}
	}

	return entries, nil
}

func buildOptOutSet() (map[string]bool, error) {
	var prefs []models.UserPreference
	if err := store.DB.Find(&prefs).Error; err != nil {
		return nil, err
	}

	optedOutUsers := make(map[uint]bool)
	for _, p := range prefs {
		parsed, err := p.ParsePreferences()
		if err != nil {
			continue
		}
		if !parsed.LeaderboardOptIn {
			optedOutUsers[p.UserID] = true
		}
	}

	if len(optedOutUsers) == 0 {
		return nil, nil
	}

	var users []models.User
	if err := store.DB.Where("whitelist_uuid IS NOT NULL AND whitelist_uuid != ''").Find(&users).Error; err != nil {
		return nil, err
	}

	exclude := make(map[string]bool)
	for _, u := range users {
		if optedOutUsers[u.ID] && u.WhitelistUUID != nil {
			exclude[*u.WhitelistUUID] = true
		}
	}

	return exclude, nil
}

func computeMetric(uuid string, metric string) (float64, bool) {
	switch metric {
	case "minecraft_playtime":
		return computeMinecraftPlaytime(uuid)
	case "achievements":
		return computeAchievements(uuid)
	case "distance":
		return computeDistance(uuid)
	case "mob_kills":
		return computeMobKills(uuid)
	case "blocks_mined":
		return computeBlocksMined(uuid)
	case "avg_move_distance":
		return computeAvgMoveDistance(uuid)
	case "login_days":
		return computeLoginDays(uuid)
	case "join_streak":
		return computeJoinStreak(uuid)
	}
	return 0, false
}

func computeMinecraftPlaytime(uuid string) (float64, bool) {
	v, err := playerdata.ReadMinecraftPlaytime(uuid)
	if err != nil {
		return 0, false
	}
	return v, true
}

func computeAchievements(uuid string) (float64, bool) {
	progress := playerdata.ReadAdvancementProgress(uuid)
	return float64(progress.Completed), true
}

func computeDistance(uuid string) (float64, bool) {
	stats, err := playerdata.ReadStats(uuid)
	if err != nil {
		return 0, false
	}

	var data struct {
		Custom map[string]float64 `json:"minecraft:custom"`
	}
	if err := json.Unmarshal(stats, &data); err != nil {
		return 0, false
	}

	var totalCm float64
	for key, val := range data.Custom {
		if strings.HasSuffix(key, "_cm") {
			totalCm += val
		}
	}
	return totalCm / 100000, true // cm to km
}

func computeMobKills(uuid string) (float64, bool) {
	stats, err := playerdata.ReadStats(uuid)
	if err != nil {
		return 0, false
	}

	var data struct {
		Custom map[string]float64 `json:"minecraft:custom"`
	}
	if err := json.Unmarshal(stats, &data); err != nil {
		return 0, false
	}

	kills, ok := data.Custom["minecraft:mob_kills"]
	return kills, ok
}

func computeBlocksMined(uuid string) (float64, bool) {
	stats, err := playerdata.ReadStats(uuid)
	if err != nil {
		return 0, false
	}

	var data struct {
		Mined map[string]float64 `json:"minecraft:mined"`
	}
	if err := json.Unmarshal(stats, &data); err != nil {
		return 0, false
	}

	var total float64
	for _, val := range data.Mined {
		total += val
	}
	return total, true
}

func computeLoginDays(uuid string) (float64, bool) {
	name := playerdata.LookupPlayerName(uuid)
	if name == "" {
		return 0, false
	}
	dates := playerdata.QueryOnlineDates(name)
	return float64(len(dates)), true
}

func computeJoinStreak(uuid string) (float64, bool) {
	name := playerdata.LookupPlayerName(uuid)
	if name == "" {
		return 0, false
	}
	streak := playerdata.QueryJoinStreak(name)
	return float64(streak), true
}

func computeAvgMoveDistance(uuid string) (float64, bool) {
	playtime, err := playerdata.ReadMinecraftPlaytime(uuid)
	if err != nil || playtime == 0 {
		return 0, false
	}

	stats, err := playerdata.ReadStats(uuid)
	if err != nil {
		return 0, false
	}

	var data struct {
		Custom map[string]float64 `json:"minecraft:custom"`
	}
	if err := json.Unmarshal(stats, &data); err != nil {
		return 0, false
	}

	var totalCm float64
	for key, val := range data.Custom {
		if strings.HasSuffix(key, "_cm") {
			totalCm += val
		}
	}

	distanceKm := totalCm / 100000
	playtimeHours := playtime / 3600
	return distanceKm / float64(playtimeHours), true
}

// BuildRawLeaderboard computes a leaderboard for a single raw Minecraft stat key
// (e.g. "minecraft:walk_one_cm", "minecraft:jump"). It reads each player's stats
// JSON and looks up the value in minecraft:custom. Players who have opted out of
// leaderboards are excluded.
func BuildRawLeaderboard(statKey, order string) ([]Entry, error) {
	uuids, err := playerdata.ListPlayerUUIDs()
	if err != nil {
		return nil, fmt.Errorf("列出玩家失败: %w", err)
	}

	excludeUUIDs, err := buildOptOutSet()
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, uuid := range uuids {
		if excludeUUIDs[uuid] {
			continue
		}

		value, ok := computeRawStat(uuid, statKey)
		if !ok {
			continue
		}

		entries = append(entries, Entry{
			UUID:       uuid,
			PlayerName: playerdata.LookupPlayerName(uuid),
			Value:      value,
		})
	}

	desc := order != "asc"
	sort.Slice(entries, func(i, j int) bool {
		if desc {
			return entries[i].Value > entries[j].Value
		}
		return entries[i].Value < entries[j].Value
	})

	if entries == nil {
		entries = []Entry{}
	}

	return entries, nil
}

// computeRawStat reads a single stat key from the player's minecraft:custom stats.
func computeRawStat(uuid, statKey string) (float64, bool) {
	stats, err := playerdata.ReadStats(uuid)
	if err != nil {
		return 0, false
	}

	var data struct {
		Custom map[string]float64 `json:"minecraft:custom"`
	}
	if err := json.Unmarshal(stats, &data); err != nil {
		return 0, false
	}

	val, ok := data.Custom[statKey]
	return val, ok
}

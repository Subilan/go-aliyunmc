package leaderboard

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/cache"
	"github.com/Subilan/go-aliyunmc/internal/playerdata"
	"github.com/Subilan/go-aliyunmc/internal/store"
	"github.com/Subilan/go-aliyunmc/internal/store/models"
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

// entriesCacheTTL 是榜单结果缓存的有效期，需短于远端数据同步的轮询间隔。
const entriesCacheTTL = 30 * time.Second

var entriesCache = cache.New[[]Entry](entriesCacheTTL, 64)

// BuildLeaderboard computes the leaderboard for the given query
func BuildLeaderboard(q Query) ([]Entry, error) {
	if !validMetrics[q.Metric] {
		return nil, fmt.Errorf("无效的指标: %s", q.Metric)
	}

	entries, err := entriesCache.GetOrLoad("metric:"+q.Metric, func() ([]Entry, error) {
		return computeLeaderboard(q.Metric)
	})
	if err != nil {
		return nil, err
	}

	return sortEntries(entries, q.Order), nil
}

func computeLeaderboard(metric string) ([]Entry, error) {
	// login-based metrics use a batch query path
	if metric == "login_days" || metric == "join_streak" {
		return computeLoginLeaderboard(metric)
	}

	uuids, err := playerdata.ListPlayerUUIDs()
	if err != nil {
		return nil, fmt.Errorf("列出玩家失败: %w", err)
	}

	excludeUUIDs, err := buildOptOutSet()
	if err != nil {
		return nil, err
	}

	nameMap := playerdata.BuildPlayerNameMap()

	var entries []Entry
	for _, uuid := range uuids {
		if excludeUUIDs[uuid] {
			continue
		}

		value, ok := computeMetric(uuid, metric)
		if !ok {
			continue
		}

		playerName := nameMap[uuid]
		if playerName == "" {
			continue
		}

		entries = append(entries, Entry{
			UUID:       uuid,
			PlayerName: playerName,
			Value:      value,
		})
	}

	if entries == nil {
		entries = []Entry{}
	}

	return entries, nil
}

// sortEntries 返回按 order 排序的副本，不修改传入的缓存值。
func sortEntries(entries []Entry, order string) []Entry {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)

	desc := order != "asc"
	slices.SortStableFunc(sorted, func(a, b Entry) int {
		if c := cmp.Compare(a.Value, b.Value); c != 0 {
			if desc {
				return -c
			}
			return c
		}
		return cmp.Compare(a.UUID, b.UUID)
	})

	return sorted
}

// computeLoginLeaderboard handles login_days and join_streak with a single
// batch query instead of per-player SQL queries.
func computeLoginLeaderboard(metric string) ([]Entry, error) {
	uuids, err := playerdata.ListPlayerUUIDs()
	if err != nil {
		return nil, fmt.Errorf("列出玩家失败: %w", err)
	}

	excludeUUIDs, err := buildOptOutSet()
	if err != nil {
		return nil, err
	}

	nameMap := playerdata.BuildPlayerNameMap()

	// Collect eligible player names for batch query
	var playerNames []string
	uuidToName := make(map[string]string)
	for _, uuid := range uuids {
		if excludeUUIDs[uuid] {
			continue
		}
		name := nameMap[uuid]
		if name == "" {
			continue
		}
		playerNames = append(playerNames, name)
		uuidToName[uuid] = name
	}

	datesByName, err := playerdata.BatchQueryOnlineDates(playerNames)
	if err != nil {
		return nil, fmt.Errorf("查询在线数据失败: %w", err)
	}

	var entries []Entry
	for uuid, name := range uuidToName {
		dates := datesByName[name]
		var value float64
		if metric == "login_days" {
			value = float64(len(dates))
		} else {
			value = float64(playerdata.ComputeJoinStreak(dates))
		}
		entries = append(entries, Entry{
			UUID:       uuid,
			PlayerName: name,
			Value:      value,
		})
	}

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
	entries, err := entriesCache.GetOrLoad("raw:"+statKey, func() ([]Entry, error) {
		return computeRawLeaderboard(statKey)
	})
	if err != nil {
		return nil, err
	}

	return sortEntries(entries, order), nil
}

func computeRawLeaderboard(statKey string) ([]Entry, error) {
	uuids, err := playerdata.ListPlayerUUIDs()
	if err != nil {
		return nil, fmt.Errorf("列出玩家失败: %w", err)
	}

	excludeUUIDs, err := buildOptOutSet()
	if err != nil {
		return nil, err
	}

	nameMap := playerdata.BuildPlayerNameMap()

	var entries []Entry
	for _, uuid := range uuids {
		if excludeUUIDs[uuid] {
			continue
		}

		value, ok := computeRawStat(uuid, statKey)
		if !ok {
			continue
		}

		playerName := nameMap[uuid]
		if playerName == "" {
			continue
		}

		entries = append(entries, Entry{
			UUID:       uuid,
			PlayerName: playerName,
			Value:      value,
		})
	}

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

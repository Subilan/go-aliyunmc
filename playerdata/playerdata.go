package playerdata

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/Subilan/go-aliyunmc/mc"
	"github.com/Subilan/go-aliyunmc/store"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	StatsDir        = "remote_data_cache/player_stats"
	AdvancementsDir = "remote_data_cache/player_advancements"
	PlaytimeDB      = "remote_data_cache/playtime.db"
	WhitelistPath   = "remote_data_cache/whitelist.json"
)

// WhitelistEntry represents a single whitelist entry
type WhitelistEntry struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// PlaytimeInfo holds playtime statistics
type PlaytimeInfo struct {
	Playtime           int64  `json:"playtime"`
	ArtificialPlaytime int64  `json:"artificial_playtime"`
	AfkPlaytime        int64  `json:"afk_playtime"`
	LastSeen           *int64 `json:"last_seen"`
	FirstJoin          *int64 `json:"first_join"`
	JoinStreak         int    `json:"join_streak"`
}

// CategoryProgress holds per-category advancement counts
type CategoryProgress struct {
	Category  string `json:"category"`
	Completed int    `json:"completed"`
	Total     int    `json:"total"`
}

// AdvancementProgress holds overall advancement completion
type AdvancementProgress struct {
	Total      int                `json:"total"`
	Completed  int                `json:"completed"`
	Categories []CategoryProgress `json:"categories"`
}

// ReadWhitelist reads the whitelist JSON file
func ReadWhitelist() ([]WhitelistEntry, error) {
	data, err := os.ReadFile(WhitelistPath)
	if err != nil {
		return nil, err
	}
	var entries []WhitelistEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// LookupPlayerName finds the player name for a UUID from the whitelist
func LookupPlayerName(uuid string) string {
	entries, err := ReadWhitelist()
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

// ReadStats reads the raw Minecraft stats JSON for a player
func ReadStats(uuid string) (json.RawMessage, error) {
	data, err := os.ReadFile(fmt.Sprintf("%s/%s.json", StatsDir, uuid))

	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Stats       json.RawMessage `json:"stats"`
		DataVersion int             `json:"DataVersion"`
	}

	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	return wrapper.Stats, nil
}

// ReadAdvancementProgress computes advancement completion for a player
func ReadAdvancementProgress(uuid string) AdvancementProgress {
	type rawAchievement struct {
		Done bool `json:"done"`
	}

	data, err := os.ReadFile(fmt.Sprintf("%s/%s.json", AdvancementsDir, uuid))
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
		category := AdvancementCategory(key)
		categoryCompleted[category]++
	}

	for key := range mc.Advancements {
		category := AdvancementCategory(key)
		categoryTotal[category]++
	}

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

// AdvancementCategory extracts the category from an advancement key
func AdvancementCategory(key string) string {
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

// QueryPlaytime queries playtime data from the SQLite database
func QueryPlaytime(uuid string) *PlaytimeInfo {
	db, err := sql.Open("sqlite3", PlaytimeDB)
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

// playerNameMatchExpr returns a SQL expression that matches playerName
// within the comma-separated player_names column.
func playerNameMatchExpr() string {
	switch store.C.Driver {
	case "mysql":
		return "CONCAT(',', player_names, ',') LIKE CONCAT('%,', ?, ',%')"
	default:
		return "',' || player_names || ',' LIKE '%,' || ? || ',%'"
	}
}

// QueryOnlineDates returns sorted list of dates when the player was online.
func QueryOnlineDates(playerName string) []string {
	if playerName == "" {
		return nil
	}

	var dates []string
	query := fmt.Sprintf(
		"SELECT DISTINCT DATE(created_at) AS date FROM player_list_samples WHERE %s ORDER BY date DESC",
		playerNameMatchExpr(),
	)
	if err := store.DB.Raw(query, playerName).Pluck("date", &dates).Error; err != nil {
		return nil
	}

	sort.Strings(dates)
	return dates
}

// QueryLastSeen returns the most recent time the player appeared online.
func QueryLastSeen(playerName string) *time.Time {
	if playerName == "" {
		return nil
	}

	query := fmt.Sprintf(
		"SELECT MAX(created_at) FROM player_list_samples WHERE %s",
		playerNameMatchExpr(),
	)

	var lastSeen time.Time
	if err := store.DB.Raw(query, playerName).Scan(&lastSeen).Error; err != nil {
		return nil
	}
	if lastSeen.IsZero() {
		return nil
	}
	return &lastSeen
}

// QueryJoinStreak returns the number of consecutive days (counting backwards
// from today) that the player has appeared online.
func QueryJoinStreak(playerName string) int {
	dates := QueryOnlineDates(playerName)
	if len(dates) == 0 {
		return 0
	}

	dateSet := make(map[string]bool, len(dates))
	for _, d := range dates {
		dateSet[d] = true
	}

	today := time.Now()
	streak := 0
	for i := 0; ; i++ {
		d := today.AddDate(0, 0, -i)
		dateStr := d.Format("2006-01-02")
		if dateSet[dateStr] {
			streak++
		} else {
			break
		}
	}
	return streak
}

// ReadMinecraftPlaytime reads the playtime (in seconds) from the player's stats JSON
func ReadMinecraftPlaytime(uuid string) (float64, error) {
	stats, err := ReadStats(uuid)
	if err != nil {
		return 0, err
	}

	var data struct {
		Custom map[string]float64 `json:"minecraft:custom"`
	}
	if err := json.Unmarshal(stats, &data); err != nil {
		return 0, err
	}

	return data.Custom["minecraft:play_time"], nil
}

// ListPlayerUUIDs returns all UUIDs that have stats files
func ListPlayerUUIDs() ([]string, error) {
	entries, err := os.ReadDir(StatsDir)
	if err != nil {
		return nil, err
	}

	var uuids []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		uuid := strings.TrimSuffix(name, ".json")
		uuids = append(uuids, uuid)
	}
	return uuids, nil
}

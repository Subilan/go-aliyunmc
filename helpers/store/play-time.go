package store

import (
	"database/sql"
	"fmt"

	"github.com/Subilan/go-aliyunmc/config"
	_ "modernc.org/sqlite"
)

// PlayTimeData 表示玩家的游戏时间数据
type PlayTimeData struct {
	UUID               string `json:"uuid,omitempty"`
	Nickname           string `json:"nickname,omitempty"`
	PlayTime           int64  `json:"playTime,omitempty"`
	ArtificialPlayTime int64  `json:"artificialPlayTime,omitempty"`
	AfkPlayTime        int64  `json:"afkPlayTime,omitempty"`
	LastSeen           int64  `json:"lastSeen,omitempty"`
	FirstJoin          int64  `json:"firstJoin,omitempty"`
	RelativeJoinStreak int    `json:"relativeJoinStreak,omitempty"`
	AbsoluteJoinStreak int    `json:"absoluteJoinStreak,omitempty"`
}

// GetPlayTimeDB 打开 play_time.db 数据库连接
func GetPlayTimeDB() (*sql.DB, error) {
	dbPath := config.Cfg.Monitor.PlayerProfile.LocalPlayTimeDbFile
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// GetPlayTimeByUUID 根据 UUID 获取玩家的游戏时间数据
func GetPlayTimeByUUID(uuid string) (*PlayTimeData, error) {
	db, err := GetPlayTimeDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var data PlayTimeData
	err = db.QueryRow(`
		SELECT uuid, '', playtime, artificial_playtime, afk_playtime,
		       last_seen, first_join, relative_join_streak, absolute_join_streak
		FROM play_time
		WHERE uuid = ?
	`, uuid).Scan(
		&data.UUID,
		&data.Nickname,
		&data.PlayTime,
		&data.ArtificialPlayTime,
		&data.AfkPlayTime,
		&data.LastSeen,
		&data.FirstJoin,
		&data.RelativeJoinStreak,
		&data.AbsoluteJoinStreak,
	)

	if err != nil {
		return nil, err
	}

	return &data, nil
}

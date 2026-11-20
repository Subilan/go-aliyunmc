package labmodels

import (
	"strings"
	"time"
)

// PlayerListSample 表示一次在线玩家列表采样数据点。
type PlayerListSample struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `gorm:"not null;index" json:"time"`
	PlayerNames string    `gorm:"not null" json:"playerNames"`
}

// SplitPlayerNames 把采样数据中保存的逗号分隔玩家名解析为列表，忽略空白项。
func SplitPlayerNames(raw string) []string {
	if raw == "" {
		return nil
	}

	names := make([]string, 0, strings.Count(raw, ",")+1)
	for name := range strings.SplitSeq(raw, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

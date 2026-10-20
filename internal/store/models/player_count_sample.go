package models

import "time"

// PlayerListSample 表示一次在线玩家列表采样数据点。
type PlayerListSample struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `gorm:"not null;index" json:"time"`
	PlayerNames string    `gorm:"not null" json:"playerNames"`
}

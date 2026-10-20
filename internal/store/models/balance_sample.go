package models

import "time"

// BalanceSample 表示一次账户余额采样数据点。
type BalanceSample struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"not null;index" json:"time"`
	Amount    float64   `gorm:"not null" json:"amount"`
}

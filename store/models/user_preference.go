package models

import (
	"encoding/json"

	"gorm.io/gorm"
)

// UserPreference 用户偏好设置，一行对应一个用户，Data 存储 JSON
type UserPreference struct {
	gorm.Model
	UserID uint   `gorm:"uniqueIndex;not null" json:"userId"`
	Data   string `gorm:"not null" json:"-"`
}

// Preferences 用户偏好项（类型安全）
type Preferences struct {
	LeaderboardOptIn bool `json:"leaderboard_opt_in"`
}

// DefaultPreferences 默认偏好设置
var DefaultPreferences = Preferences{
	LeaderboardOptIn: true,
}

// ParsePreferences 从 JSON 字符串解析偏好，用默认值填充零值字段
func (p *UserPreference) ParsePreferences() (Preferences, error) {
	prefs := DefaultPreferences
	if p.Data != "" {
		if err := json.Unmarshal([]byte(p.Data), &prefs); err != nil {
			return DefaultPreferences, err
		}
	}
	return prefs, nil
}

// SetPreferences 将偏好序列化为 JSON 存入 Data
func (p *UserPreference) SetPreferences(prefs Preferences) error {
	b, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	p.Data = string(b)
	return nil
}

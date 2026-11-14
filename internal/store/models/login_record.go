package models

import "time"

// LoginRecord 记录一次成功登录。
type LoginRecord struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// UserID 是登录用户的 ID。
	UserID uint `gorm:"not null;index:idx_login_records_user_created,priority:1" json:"user_id"`
	// CreatedAt 是登录成功的时间。
	CreatedAt time.Time `gorm:"not null;index:idx_login_records_user_created,priority:2" json:"created_at"`
}

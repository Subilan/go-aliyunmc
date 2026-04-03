package models

import (
	"gorm.io/gorm"
)

// User 是用户模型
type User struct {
	gorm.Model
	DeletedAt gorm.DeletedAt `gorm:"uniqueIndex:idx_username_deleted_at"`
	// Username 是用户名
	Username string `gorm:"uniqueIndex:idx_username_deleted_at" json:"username"`
	// PasswordHash 是密码哈希
	PasswordHash string `gorm:"not null" json:"-"`
	// Role 是用户角色
	Role string `gorm:"default:'basic'" json:"role"`
}

package models

import "gorm.io/gorm"

// User 是用户模型
type User struct {
	gorm.Model
	// Username 是用户名
	Username string `gorm:"unique" json:"username"`
	// Password 是密码
	Password string `gorm:"not null" json:"-"`
	// PasswordHash 是密码哈希
	PasswordHash string `gorm:"not null" json:"-"`
	// Role 是用户角色
	Role string `gorm:"default:'basic'" json:"role"`
}

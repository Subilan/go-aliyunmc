package models

import (
	"github.com/Subilan/go-aliyunmc/perms"

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
	Role perms.Role `gorm:"default:''" json:"role"`
	// Banned 标记用户是否已被封禁
	Banned bool `gorm:"default:false" json:"banned"`
	// WhitelistUUID 是绑定的白名单项目UUID，一对一关系
	WhitelistUUID *string `gorm:"uniqueIndex:idx_whitelist_uuid" json:"whitelist_uuid,omitempty"`
	// LikedChangelogs 是用户点赞过的 changelog
	LikedChangelogs []Changelog `gorm:"many2many:changelog_likes;" json:"-"`
}

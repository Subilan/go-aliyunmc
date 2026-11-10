package models

import (
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"

	"gorm.io/gorm"
)

// User 是共享用户模型，lab 与 forum 共用同一份账号与凭据。
type User struct {
	gorm.Model
	DeletedAt gorm.DeletedAt `gorm:"uniqueIndex:idx_username_deleted_at"`
	// Username 是用户名。size:191 保证 utf8mb4 下可以建立唯一索引（MySQL）。
	Username string `gorm:"size:191;uniqueIndex:idx_username_deleted_at" json:"username"`
	// PasswordHash 是密码哈希
	PasswordHash string `gorm:"not null" json:"-"`
	// Role 是用户角色。
	Role perm_util.Role `gorm:"default:''" json:"role"`
	// Banned 标记用户是否已被封禁
	Banned bool `gorm:"default:false" json:"banned"`
	// WhitelistUUID 是绑定的白名单项目UUID，一对一关系
	WhitelistUUID *string `gorm:"size:191;uniqueIndex:idx_whitelist_uuid" json:"whitelist_uuid,omitempty"`
}

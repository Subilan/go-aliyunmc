package forummodels

import "time"

// UserPermission 是直接授予用户的权限节点。ScopeBoardID 为 0 表示全局。
type UserPermission struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index:idx_forum_user_permissions_user_id;uniqueIndex:idx_forum_user_permissions_unique,priority:1" json:"userId"`
	PermissionCode string    `gorm:"size:128;not null;index:idx_forum_user_permissions_code;uniqueIndex:idx_forum_user_permissions_unique,priority:2" json:"permissionCode"`
	ScopeBoardID   uint      `gorm:"not null;default:0;uniqueIndex:idx_forum_user_permissions_unique,priority:3" json:"scopeBoardId"`
	GrantedBy      uint      `gorm:"not null" json:"grantedBy"`
	CreatedAt      time.Time `gorm:"not null" json:"createdAt"`
}

func (UserPermission) TableName() string { return "forum_user_permissions" }

// Setting 是站点设置键值。默认值在代码中，数据库只保存被修改过的键。
type Setting struct {
	Key       string    `gorm:"size:64;primaryKey;not null" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	UpdatedBy uint      `gorm:"not null" json:"updatedBy"`
	UpdatedAt time.Time `gorm:"not null" json:"updatedAt"`
}

func (Setting) TableName() string { return "forum_settings" }

// AuditLog 是管理操作审计。Detail 不记录正文与密码。
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ActorID    uint      `gorm:"not null;index:idx_forum_audit_logs_actor_id" json:"actorId"`
	Action     string    `gorm:"size:64;not null;index:idx_forum_audit_logs_action" json:"action"`
	TargetType string    `gorm:"size:32;not null" json:"targetType"`
	TargetID   uint      `gorm:"not null;default:0" json:"targetId"`
	Detail     string    `gorm:"type:text;not null" json:"detail"`
	IP         string    `gorm:"size:45;not null;default:''" json:"ip"`
	UserAgent  string    `gorm:"size:255;not null;default:''" json:"userAgent"`
	CreatedAt  time.Time `gorm:"not null;index:idx_forum_audit_logs_created_at" json:"createdAt"`
}

func (AuditLog) TableName() string { return "forum_audit_logs" }

// LoginChannel 是登录渠道。
type LoginChannel string

const (
	LoginChannelForum LoginChannel = "forum"
	LoginChannelAdmin LoginChannel = "admin"
)

// LoginRecord 是论坛与后台的登录记录。
type LoginRecord struct {
	ID        uint         `gorm:"primaryKey" json:"id"`
	UserID    uint         `gorm:"not null;index:idx_forum_login_records_user_id" json:"userId"`
	Channel   LoginChannel `gorm:"size:16;not null" json:"channel"`
	IP        string       `gorm:"size:45;not null;default:''" json:"ip"`
	UserAgent string       `gorm:"size:255;not null;default:''" json:"userAgent"`
	CreatedAt time.Time    `gorm:"not null;index:idx_forum_login_records_created_at" json:"createdAt"`
}

func (LoginRecord) TableName() string { return "forum_login_records" }

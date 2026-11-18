package forummodels

import "gorm.io/gorm"

// AvatarShape 是头像显示形状。
type AvatarShape string

const (
	AvatarShapeCircle AvatarShape = "circle"
	AvatarShapeSquare AvatarShape = "square"
)

// Profile 是论坛用户资料，与共享 users 一对一。论坛从不删除资料行。
type Profile struct {
	gorm.Model
	UserID uint `gorm:"not null;uniqueIndex:idx_forum_profiles_user_id" json:"userId"`
	// Nickname 是可选的昵称；为空时前台展示 users.username。
	Nickname *string `gorm:"size:32" json:"nickname"`
	// Bio 是纯文本简介。
	Bio string `gorm:"size:200;not null;default:''" json:"bio"`
	// AvatarUploadID 指向 forum_uploads，为空时使用默认头像。
	AvatarUploadID *uint `gorm:"index:idx_forum_profiles_avatar_upload_id" json:"avatarUploadId"`
	// BackgroundUploadID 指向论坛主页背景上传。
	BackgroundUploadID *uint `gorm:"index:idx_forum_profiles_background_upload_id" json:"backgroundUploadId"`
	// AvatarShape 是 circle 或 square。
	AvatarShape AvatarShape `gorm:"size:8;not null;default:'circle'" json:"avatarShape"`
	// PermissionsInitialized 记录是否已经执行过首次登录的默认权限发放。
	PermissionsInitialized bool `gorm:"not null;default:false" json:"permissionsInitialized"`
}

func (Profile) TableName() string { return "forum_profiles" }

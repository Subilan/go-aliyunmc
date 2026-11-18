package forummodels

import "gorm.io/gorm"

// UploadKind 是上传文件的用途。
type UploadKind string

const (
	UploadKindPost       UploadKind = "post"
	UploadKindAvatar     UploadKind = "avatar"
	UploadKindBackground UploadKind = "background"
)

// Upload 是上传文件的元数据。文件本身存放在本地磁盘。
type Upload struct {
	gorm.Model
	OwnerID      uint       `gorm:"not null;index:idx_forum_uploads_owner_id" json:"ownerId"`
	Kind         UploadKind `gorm:"size:16;not null" json:"kind"`
	Key          string     `gorm:"size:255;not null;uniqueIndex:idx_forum_uploads_key" json:"key"`
	OriginalName string     `gorm:"size:255;not null;default:''" json:"originalName"`
	Mime         string     `gorm:"size:64;not null" json:"mime"`
	SizeBytes    int64      `gorm:"not null" json:"sizeBytes"`
	Width        int        `gorm:"not null" json:"width"`
	Height       int        `gorm:"not null" json:"height"`
	SHA256       string     `gorm:"type:char(64);not null;index:idx_forum_uploads_sha256" json:"sha256"`
}

func (Upload) TableName() string { return "forum_uploads" }

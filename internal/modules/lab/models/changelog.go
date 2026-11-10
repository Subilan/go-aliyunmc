package labmodels

import (
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"

	"gorm.io/gorm"
)

// LogType 表示 changelog 的分类
type LogType string

const (
	LogTypePlatform LogType = "platform"
	LogTypeServer   LogType = "server"
)

// Changelog 表示一条系统更新日志
type Changelog struct {
	gorm.Model
	Title    string        `gorm:"not null" json:"title"`
	Body     string        `gorm:"not null" json:"body"`
	Category LogType       `gorm:"not null;index" json:"category"`
	LikedBy  []models.User `gorm:"many2many:changelog_likes;" json:"-"`
}

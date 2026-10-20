package store

import (
	"github.com/Subilan/go-aliyunmc/internal/env"
	"github.com/Subilan/go-aliyunmc/internal/store/models"
)

// AutoMigrate 在 DEV 模式按 GORM 模型补齐数据库表结构。
func AutoMigrate() error {
	if !env.DEV {
		return nil
	}

	return DB.AutoMigrate(models.Entities()...)
}

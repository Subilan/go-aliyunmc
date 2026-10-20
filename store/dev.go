package store

import (
	"github.com/Subilan/go-aliyunmc/env"
	"github.com/Subilan/go-aliyunmc/store/models"
)

// AutoMigrate 在 DEV 模式按 GORM 模型补齐数据库表结构。
func AutoMigrate() error {
	if !env.DEV {
		return nil
	}

	return DB.AutoMigrate(models.Entities()...)
}

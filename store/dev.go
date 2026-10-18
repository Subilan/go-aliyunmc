package store

import (
	"github.com/Subilan/go-aliyunmc/env"
	"github.com/Subilan/go-aliyunmc/store/models"
)

// AutoMigrate 在 DEV 模式按 GORM 模型补齐数据库表结构。
//
// 生产环境不使用 AutoMigrate，schema 变更必须走 `migrate` 的版本化 SQL；
// 开发环境允许直接按模型补表，省去每次改模型都生成迁移的步骤。
func AutoMigrate() error {
	if !env.DEV {
		return nil
	}

	return DB.AutoMigrate(models.Entities()...)
}

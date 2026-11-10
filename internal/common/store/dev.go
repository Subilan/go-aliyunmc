package store

import "github.com/Subilan/go-aliyunmc/internal/common/env"

// AutoMigrate 在 DEV 模式按 GORM 模型补齐数据库表结构。
// entities 由调用方提供（通常使用 internal/modules.Entities 合并后的完整列表）。
func AutoMigrate(entities ...any) error {
	if !env.DEV {
		return nil
	}

	return DB.AutoMigrate(entities...)
}

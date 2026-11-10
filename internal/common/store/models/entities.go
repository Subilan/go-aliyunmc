package models

// Entities 返回共享层参与数据库迁移的 GORM 实体。
func Entities() []any {
	return []any{
		&User{},
	}
}

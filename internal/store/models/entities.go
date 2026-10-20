package models

// Entities 返回所有参与数据库迁移的 GORM 实体。
func Entities() []any {
	return []any{
		&User{},
		&Task{},
		&TaskOutputLine{},
		&Instance{},
		&PlayerListSample{},
		&BalanceSample{},
		&UserPreference{},
		&Changelog{},
	}
}

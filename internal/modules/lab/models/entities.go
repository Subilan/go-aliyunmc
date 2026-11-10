package labmodels

// Entities 返回 lab 模块参与数据库迁移的 GORM 实体。
// 共享的 User 模型由 common/store/models.Entities 提供，二者由 internal/modules.Entities 合并。
func Entities() []any {
	return []any{
		&Task{},
		&TaskOutputLine{},
		&Instance{},
		&PlayerListSample{},
		&BalanceSample{},
		&UserPreference{},
		&Changelog{},
	}
}

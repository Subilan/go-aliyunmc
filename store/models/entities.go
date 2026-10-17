package models

// Entities 返回所有参与数据库迁移的 GORM 实体。
//
// tools/atlasloader 使用该列表生成迁移，因此新增模型时只需修改这里
// （以及对应迁移的生成流程）。
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

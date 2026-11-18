// Package forummodels 定义论坛模块的 GORM 实体。
// 所有表使用 forum_ 前缀，与 lab 共用同一个数据库。
package forummodels

// Entities 返回论坛模块参与数据库迁移的 GORM 实体。
func Entities() []any {
	return []any{
		&Profile{},
		&Board{},
		&Thread{},
		&Reply{},
		&Tag{},
		&ThreadTag{},
		&ThreadLike{},
		&ReplyLike{},
		&UserLike{},
		&BoardFollow{},
		&Notification{},
		&Upload{},
		&UserPermission{},
		&Setting{},
		&AuditLog{},
		&LoginRecord{},
	}
}

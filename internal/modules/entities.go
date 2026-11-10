// Package modules 汇总各业务模块参与数据库迁移的实体。
// 共享层（common）不能 import 业务模块，因此由本包负责合并。
package modules

import (
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"
)

// Entities 返回共享层与所有业务模块的 GORM 实体。
func Entities() []any {
	entities := models.Entities()
	return append(entities, labmodels.Entities()...)
}

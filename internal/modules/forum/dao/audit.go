package dao

import (
	"encoding/json"
	"fmt"

	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/str_util"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	"github.com/gin-gonic/gin"
)

// maxAuditDetailBytes 是审计 detail 的字节上限，避免超出 TEXT 列并保证 JSON 合法。
const maxAuditDetailBytes = 32 * 1024

// Audit 写入一条管理操作审计。
func Audit(c *gin.Context, actorID uint, action, targetType string, targetID uint, detail any) {
	payload := "{}"
	if detail != nil {
		if data, err := json.Marshal(detail); err == nil {
			payload = string(data)
		}
	}
	if len(payload) > maxAuditDetailBytes {
		payload = fmt.Sprintf(`{"truncated":true,"bytes":%d}`, len(payload))
	}

	ip := c.ClientIP()
	userAgent := str_util.Truncate(c.Request.UserAgent(), 255)

	entry := forummodels.AuditLog{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     payload,
		IP:         ip,
		UserAgent:  userAgent,
	}
	if err := store.DB.Create(&entry).Error; err != nil {
		// 审计失败不阻断主流程，但必须留下显眼的错误日志。
		log_util.Error("写入论坛审计日志失败: actor=%d action=%s error=%v", actorID, action, err)
	}
}

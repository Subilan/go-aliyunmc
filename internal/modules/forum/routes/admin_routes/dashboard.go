package admin_routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

func bindDashboard(admin *gin.RouterGroup) {
	admin.GET("/dashboard", mid.RequirePermission("stats.view"), h.G(dashboard))
}

func dashboard(c *gin.Context) (any, error) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	return service.AdminDashboard(days)
}

func bindAudit(admin *gin.RouterGroup) {
	admin.GET("/audit-logs", mid.RequirePermission("audit.view"), h.G(auditLogs))
}

func auditLogs(c *gin.Context) (any, error) {
	actorID, _ := strconv.ParseUint(c.DefaultQuery("actorId", "0"), 10, 64)
	targetID, _ := strconv.ParseUint(c.DefaultQuery("targetId", "0"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var from, to *time.Time
	if value := c.Query("from"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, h.HttpError(http.StatusBadRequest, "from 必须是 RFC3339 时间")
		}
		from = &parsed
	}
	if value := c.Query("to"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, h.HttpError(http.StatusBadRequest, "to 必须是 RFC3339 时间")
		}
		to = &parsed
	}

	return service.AdminAuditLogs(
		uint(actorID),
		c.Query("action"),
		c.Query("targetType"),
		uint(targetID),
		from,
		to,
		page,
		pageSize,
	)
}

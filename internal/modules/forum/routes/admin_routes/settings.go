package admin_routes

import (
	"strconv"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

func bindSettings(admin *gin.RouterGroup) {
	admin.GET("/settings", mid.RequirePermission("settings.manage"), h.G(getSettings))
	admin.PUT("/settings", mid.RequirePermission("settings.manage"), h.B(updateSettings))
}

func getSettings(c *gin.Context) (any, error) {
	return service.AdminGetSettings(), nil
}

func updateSettings(updates map[string]string, c *gin.Context) (any, error) {
	actor, _ := mid.CurrentUser(c)
	return service.AdminUpdateSettings(actor, updates, c)
}

func bindUploads(admin *gin.RouterGroup) {
	admin.GET("/uploads", mid.RequirePermission("upload.manage"), h.G(listUploads))
	admin.GET("/uploads/stats", mid.RequirePermission("upload.manage"), h.G(uploadStats))
	admin.DELETE("/uploads/:id", mid.RequirePermission("upload.manage"), h.G(deleteUpload))
}

func listUploads(c *gin.Context) (any, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	ownerID, _ := strconv.ParseUint(c.DefaultQuery("ownerId", "0"), 10, 64)
	return service.AdminListUploads(c.Query("kind"), uint(ownerID), page, pageSize)
}

func uploadStats(c *gin.Context) (any, error) {
	items, err := service.AdminUploadStats()
	if err != nil {
		return nil, err
	}
	return gin.H{"items": items}, nil
}

func deleteUpload(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	actor, _ := mid.CurrentUser(c)
	if err := service.AdminDeleteUpload(actor, id, c); err != nil {
		return nil, err
	}
	return nil, nil
}

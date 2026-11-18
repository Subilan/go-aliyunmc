package admin_routes

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/routes/routeutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/service"

	"github.com/gin-gonic/gin"
)

type grantsRequest struct {
	Grants []perms.Grant `json:"grants"`
}

// adminProfilePatch 用 RawMessage 区分“未提供”与“显式置空”。
type adminProfilePatch struct {
	Nickname json.RawMessage `json:"nickname"`
	Bio      json.RawMessage `json:"bio"`
}

func bindUsers(admin *gin.RouterGroup) {
	admin.GET("/users", mid.RequirePermission("user.list"), h.G(listUsers))
	admin.GET("/users/:id", mid.RequirePermission("user.list"), h.G(userDetail))
	admin.PATCH("/users/:id/profile", mid.RequirePermission("user.manage"), h.B(updateUserProfile))
	admin.POST("/users/:id/ban", mid.RequirePermission("user.ban"), h.G(banUser))
	admin.POST("/users/:id/unban", mid.RequirePermission("user.ban"), h.G(unbanUser))
	admin.GET("/users/:id/permissions", mid.RequireAnyPermission("permission.view", "permission.manage"), h.G(userPermissions))
	admin.PUT("/users/:id/permissions", mid.RequirePermission("permission.manage"), h.B(replaceUserPermissions))
	admin.GET("/users/:id/login-records", mid.RequirePermission("user.list"), h.G(loginRecords))
}

func listUsers(c *gin.Context) (any, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	banned, err := optionalBool(c.Query("banned"))
	if err != nil {
		return nil, h.HttpError(http.StatusBadRequest, "banned 必须是布尔值")
	}

	return service.AdminListUsers(c.Query("keyword"), banned, c.Query("permission"), page, pageSize)
}

func userDetail(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	return service.AdminUserDetail(id)
}

func updateUserProfile(req adminProfilePatch, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}

	update := service.ProfileUpdate{}
	if req.Nickname != nil {
		update.NicknameSet = true
		if string(req.Nickname) != "null" {
			var nickname string
			if err := json.Unmarshal(req.Nickname, &nickname); err != nil {
				return nil, h.HttpError(http.StatusBadRequest, "nickname 必须是字符串或 null")
			}
			update.Nickname = &nickname
		}
	}
	if req.Bio != nil {
		update.BioSet = true
		if err := json.Unmarshal(req.Bio, &update.Bio); err != nil {
			return nil, h.HttpError(http.StatusBadRequest, "bio 必须是字符串")
		}
	}

	actor, _ := mid.CurrentUser(c)
	if err := service.AdminUpdateUserProfile(actor, id, update, c); err != nil {
		return nil, err
	}
	return service.AdminUserDetail(id)
}

func banUser(c *gin.Context) (any, error) {
	return setBanned(c, true)
}

func unbanUser(c *gin.Context) (any, error) {
	return setBanned(c, false)
}

func setBanned(c *gin.Context, banned bool) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	actor, _ := mid.CurrentUser(c)
	if err := service.AdminBanUser(actor, id, banned, c); err != nil {
		return nil, err
	}
	return service.AdminUserDetail(id)
}

func userPermissions(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	grants, err := service.AdminUserPermissions(id)
	if err != nil {
		return nil, err
	}
	return gin.H{"items": grants}, nil
}

func replaceUserPermissions(req grantsRequest, c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	actor, _ := mid.CurrentUser(c)
	grants, err := service.AdminReplacePermissions(actor, id, req.Grants, c)
	if err != nil {
		return nil, err
	}
	return gin.H{"items": grants}, nil
}

func loginRecords(c *gin.Context) (any, error) {
	id, err := routeutil.UintParam(c, "id")
	if err != nil {
		return nil, err
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	return service.AdminLoginRecords(id, page, pageSize)
}

func bindPermissions(admin *gin.RouterGroup) {
	admin.GET("/permissions", mid.RequirePermission("permission.view"), h.G(permissionCatalog))
	admin.GET("/permissions/:code/users", mid.RequirePermission("permission.view"), h.G(permissionUsers))
}

func permissionCatalog(c *gin.Context) (any, error) {
	return gin.H{"items": service.AdminPermissionCatalog()}, nil
}

func permissionUsers(c *gin.Context) (any, error) {
	code := c.Param("code")

	var boardID *uint
	if value := c.Query("boardId"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, h.HttpError(http.StatusBadRequest, "boardId 必须是数字")
		}
		id := uint(parsed)
		boardID = &id
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	return service.AdminPermissionUsers(code, boardID, page, pageSize)
}

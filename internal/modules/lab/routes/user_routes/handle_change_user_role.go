package user_routes

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/context_util"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"

	"github.com/gin-gonic/gin"
)

// ChangeUserRoleRequest 修改用户权限请求。
type ChangeUserRoleRequest struct {
	UserID uint           `json:"user_id"`
	Role   perm_util.Role `json:"role"`
}

// HandleChangeUserRole 修改指定用户的权限，不允许修改自己的权限。
func HandleChangeUserRole(req ChangeUserRoleRequest, c *gin.Context) (any, error) {
	// 登录与固定路由权限已由 Auth、Perm 中间件校验。
	currentUser, _ := context_util.GetUser(c)

	if req.UserID == 0 {
		return nil, h.HttpError(http.StatusBadRequest, "缺少用户 ID")
	}

	if req.UserID == currentUser.ID {
		return nil, h.HttpError(http.StatusForbidden, "不能变更自己的权限")
	}

	if !req.Role.IsValid() {
		return nil, h.HttpError(http.StatusBadRequest, "无效的权限等级")
	}

	var target models.User
	if err := store.DB.First(&target, req.UserID).Error; err != nil {
		return nil, err
	}

	target.Role = req.Role
	if err := store.DB.Save(&target).Error; err != nil {
		return nil, err
	}

	return target, nil
}

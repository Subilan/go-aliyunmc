package user_routes

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/context_util"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/perms"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
)

// BanRequest 封禁/解封请求
type BanRequest struct {
	UserIDs   []uint   `json:"user_ids"`
	Usernames []string `json:"usernames"`
}

// BanResponse 封禁/解封响应
type BanResponse struct {
	AffectedCount int      `json:"affected_count"`
	NotFoundIDs   []uint   `json:"not_found_ids,omitempty"`
	NotFoundNames []string `json:"not_found_usernames,omitempty"`
	SkippedSelf   bool     `json:"skipped_self,omitempty"`
}

// banUsers 内部封禁/解封实现
func banUsers(req BanRequest, c *gin.Context, banned bool) (any, error) {
	currentUser, _ := context_util.GetUser(c)
	currentRole, _ := context_util.GetUserRole(c)

	var affectedCount int
	var notFoundIDs []uint
	var notFoundNames []string
	var skippedSelf bool

	// 按用户ID操作
	for _, id := range req.UserIDs {
		if id == currentUser.ID {
			skippedSelf = true
			continue
		}
		var user models.User
		if err := store.DB.First(&user, id).Error; err != nil {
			notFoundIDs = append(notFoundIDs, id)
			continue
		}
		// 角色检查：只能操作角色等级低于自己的用户
		if !currentRole.Gt(perms.Role(user.Role)) {
			continue
		}
		if user.Banned == banned {
			continue // 已经是目标状态，跳过
		}
		user.Banned = banned
		store.DB.Save(&user)
		affectedCount++
	}

	// 按用户名操作
	for _, username := range req.Usernames {
		if username == currentUser.Username {
			skippedSelf = true
			continue
		}
		var user models.User
		if err := store.DB.Where("username = ?", username).First(&user).Error; err != nil {
			notFoundNames = append(notFoundNames, username)
			continue
		}
		if !currentRole.Gt(perms.Role(user.Role)) {
			continue
		}
		if user.Banned == banned {
			continue
		}
		user.Banned = banned
		store.DB.Save(&user)
		affectedCount++
	}

	if affectedCount == 0 && len(notFoundIDs) == 0 && len(notFoundNames) == 0 && !skippedSelf {
		return nil, h.HttpError(http.StatusBadRequest, "没有可操作的用户（用户不存在、角色权限不足或已是目标状态）")
	}

	return BanResponse{
		AffectedCount: affectedCount,
		NotFoundIDs:   notFoundIDs,
		NotFoundNames: notFoundNames,
		SkippedSelf:   skippedSelf,
	}, nil
}

// HandleBanUser 封禁用户
func HandleBanUser(req BanRequest, c *gin.Context) (any, error) {
	return banUsers(req, c, true)
}

// HandleUnbanUser 解除封禁用户
func HandleUnbanUser(req BanRequest, c *gin.Context) (any, error) {
	return banUsers(req, c, false)
}

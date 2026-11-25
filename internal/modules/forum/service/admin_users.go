package service

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminListUsers 返回已登录过论坛前台的共享用户 + 论坛资料列表。
// 从未登录过论坛的 lab 账号在论坛侧视为不存在。
func AdminListUsers(keyword string, banned *bool, permissions []string, page, pageSize int) (dto.Page[dto.AdminUser], error) {
	db := store.DB.Model(&models.User{}).
		Joins("LEFT JOIN forum_profiles ON forum_profiles.user_id = users.id AND forum_profiles.deleted_at IS NULL").
		Where("users.deleted_at IS NULL").
		Where(dao.ForumLoginExistsSQL, forummodels.LoginChannelForum)

	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("users.username LIKE ? OR forum_profiles.nickname LIKE ?", like, like)
	}
	if banned != nil {
		db = db.Where("users.banned = ?", *banned)
	}
	if len(permissions) > 0 {
		db = db.Where("EXISTS (SELECT 1 FROM forum_user_permissions up WHERE up.user_id = users.id AND up.permission_code IN ?)", permissions)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dto.Page[dto.AdminUser]{}, err
	}

	page, pageSize = dao.Pagination(page, pageSize, 20, 100)

	var users []models.User
	if err := db.Select("users.*").Order("users.id ASC").
		Offset(dao.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		return dto.Page[dto.AdminUser]{}, err
	}

	return dto.NewPage(adminUsers(users), total, page, pageSize), nil
}

// AdminUserDetail 返回用户详情。
func AdminUserDetail(userID uint) (*dto.AdminUser, error) {
	var user models.User
	if err := store.DB.First(&user, userID).Error; err != nil {
		return nil, h.HttpError(http.StatusNotFound, "用户不存在")
	}
	if err := requireForumParticipant(userID); err != nil {
		return nil, err
	}

	items := adminUsers([]models.User{user})
	return &items[0], nil
}

// adminUsers 批量组装后台用户 DTO，避免逐用户查询计数、登录记录与权限。
func adminUsers(users []models.User) []dto.AdminUser {
	ids := make([]uint, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}

	views := dao.LoadUserViews(ids)
	counts := dao.UserStatsBatch(ids)
	logins := dao.LastLoginTimes(ids)
	grants := perms.GrantsForMany(users)

	items := make([]dto.AdminUser, 0, len(users))
	for _, user := range users {
		login := logins[user.ID]
		count := counts[user.ID]
		items = append(items, dto.AdminUserFrom(
			user,
			views[user.ID],
			count.Threads,
			count.Replies,
			count.Likes,
			login.Forum,
			login.Admin,
			grants[user.ID],
		))
	}
	return items
}

// AdminUpdateUserProfile 修改用户资料。
func AdminUpdateUserProfile(actor *models.User, userID uint, update ProfileUpdate, c *gin.Context) error {
	var target models.User
	if err := store.DB.First(&target, userID).Error; err != nil {
		return h.HttpError(http.StatusNotFound, "用户不存在")
	}
	if err := requireForumParticipant(userID); err != nil {
		return err
	}

	if err := UpdateProfile(&target, update); err != nil {
		return err
	}

	dao.Audit(c, actor.ID, "user.profile.update", "user", target.ID, map[string]any{
		"nicknameSet": update.NicknameSet,
		"bioSet":      update.BioSet,
	})
	return nil
}

// AdminBanUser 封禁/解封用户。
// 非超级用户不能封禁超级用户，避免破窗账号被委派管理员锁死。
func AdminBanUser(actor *models.User, userID uint, banned bool, c *gin.Context) error {
	if actor.ID == userID {
		return h.HttpError(http.StatusForbidden, "不能封禁自己")
	}

	var target models.User
	if err := store.DB.First(&target, userID).Error; err != nil {
		return h.HttpError(http.StatusNotFound, "用户不存在")
	}
	if err := requireForumParticipant(userID); err != nil {
		return err
	}

	if banned && !perms.IsSuperuser(actor) && perms.IsSuperuser(&target) {
		return h.HttpError(http.StatusForbidden, "不能封禁超级用户")
	}

	if err := store.DB.Model(&models.User{}).Where("id = ?", userID).Update("banned", banned).Error; err != nil {
		return err
	}

	action := "user.unban"
	if banned {
		action = "user.ban"
	}
	dao.Audit(c, actor.ID, action, "user", userID, nil)
	return nil
}

// AdminUserPermissions 返回用户当前的权限节点。
func AdminUserPermissions(userID uint) ([]perms.Grant, error) {
	var user models.User
	if err := store.DB.First(&user, userID).Error; err != nil {
		return nil, h.HttpError(http.StatusNotFound, "用户不存在")
	}

	return perms.GrantsOf(&user), nil
}

// maxGrants 限制单次替换的权限节点数量。
const maxGrants = 500

// AdminReplacePermissions 用请求内容完整替换用户的权限节点。
func AdminReplacePermissions(actor *models.User, targetID uint, grants []perms.Grant, c *gin.Context) ([]perms.Grant, error) {
	if grants == nil {
		return nil, h.HttpError(http.StatusBadRequest, "缺少 grants 字段")
	}
	if len(grants) > maxGrants {
		return nil, h.HttpError(http.StatusBadRequest, "权限节点数量超过上限")
	}

	var target models.User
	if err := store.DB.First(&target, targetID).Error; err != nil {
		return nil, h.HttpError(http.StatusNotFound, "用户不存在")
	}

	normalized, err := normalizeGrants(grants)
	if err != nil {
		return nil, err
	}

	oldSet := grantSet(perms.GrantsOf(&target))
	newSet := grantSet(normalized)

	if !perms.IsSuperuser(actor) {
		for key := range newSet {
			if !oldSet[key] && !operatorOwnsGrant(actor, key) {
				return nil, h.HttpError(http.StatusForbidden, "不能授予自己未持有的权限节点: "+key.code)
			}
		}
		for key := range oldSet {
			if !newSet[key] && !operatorOwnsGrant(actor, key) {
				return nil, h.HttpError(http.StatusForbidden, "不能撤销自己未持有的权限节点: "+key.code)
			}
		}
	}

	if target.ID == actor.ID {
		if err := checkSelfLockout(actor, oldSet, newSet); err != nil {
			return nil, err
		}
	}

	err = store.DB.Transaction(func(tx *gorm.DB) error {
		profile, _, err := dao.EnsureProfile(tx, target.ID)
		if err != nil {
			return err
		}

		if err := perms.ReplaceGrants(tx, target.ID, normalized, actor.ID); err != nil {
			return err
		}

		return tx.Model(&forummodels.Profile{}).
			Where("id = ?", profile.ID).
			Update("permissions_initialized", true).Error
	})
	if err != nil {
		return nil, err
	}

	if err := perms.Reload(); err != nil {
		return nil, err
	}

	dao.Audit(c, actor.ID, "permission.replace", "user", target.ID, map[string]any{"grants": normalized})
	return normalized, nil
}

type grantKey struct {
	code    string
	boardID uint
}

func grantSet(grants []perms.Grant) map[grantKey]bool {
	set := make(map[grantKey]bool, len(grants))
	for _, grant := range grants {
		key := grantKey{code: grant.PermissionCode}
		if grant.BoardID != nil {
			key.boardID = *grant.BoardID
		}
		set[key] = true
	}
	return set
}

func operatorOwnsGrant(actor *models.User, key grantKey) bool {
	if key.boardID == 0 {
		return perms.Allow(actor, "*", key.code)
	}
	return perms.HasBoardPermission(actor, key.boardID, key.code)
}

func normalizeGrants(grants []perms.Grant) ([]perms.Grant, error) {
	seen := make(map[grantKey]bool, len(grants))
	result := make([]perms.Grant, 0, len(grants))

	for _, grant := range grants {
		definition, ok := perms.Get(grant.PermissionCode)
		if !ok {
			return nil, h.HttpError(http.StatusBadRequest, "未知权限点: "+grant.PermissionCode)
		}

		var boardID *uint
		if grant.BoardID != nil && *grant.BoardID != 0 {
			boardID = grant.BoardID
			if definition.Scope != perms.ScopeBoard {
				return nil, h.HttpError(http.StatusBadRequest, "权限点 "+grant.PermissionCode+" 只能全局授予")
			}
			if _, err := dao.BoardByID(*boardID, true); err != nil {
				return nil, h.HttpError(http.StatusBadRequest, "板块不存在")
			}
		}

		key := grantKey{code: grant.PermissionCode}
		if boardID != nil {
			key.boardID = *boardID
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, perms.Grant{PermissionCode: grant.PermissionCode, BoardID: boardID})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].PermissionCode != result[j].PermissionCode {
			return result[i].PermissionCode < result[j].PermissionCode
		}
		return boardValue(result[i].BoardID) < boardValue(result[j].BoardID)
	})

	return result, nil
}

func boardValue(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

// checkSelfLockout 防止管理员撤销自己的关键权限导致后台锁死。
func checkSelfLockout(actor *models.User, oldSet, newSet map[grantKey]bool) error {
	if perms.IsSuperuser(actor) {
		// 超级用户走破窗，不依赖权限记录。
		return nil
	}

	hasAdminAccess := false
	for key := range newSet {
		if key.code == "admin.access" {
			hasAdminAccess = true
			break
		}
	}
	if !hasAdminAccess {
		return h.HttpError(http.StatusBadRequest, "不能撤销自己全部的 admin.access")
	}

	for _, code := range []string{"admin.access", "permission.manage"} {
		oldGlobal := oldSet[grantKey{code: code}]
		newGlobal := newSet[grantKey{code: code}]
		if !oldGlobal || newGlobal {
			continue
		}

		var count int64
		err := store.DB.Model(&forummodels.UserPermission{}).
			Joins("JOIN users ON users.id = forum_user_permissions.user_id AND users.deleted_at IS NULL AND users.role <> ?", perm_util.RoleSuperuser).
			Where("forum_user_permissions.permission_code = ? AND forum_user_permissions.scope_board_id = 0 AND forum_user_permissions.user_id <> ?", code, actor.ID).
			Count(&count).Error
		if err != nil {
			return err
		}
		if count == 0 {
			return h.HttpError(http.StatusBadRequest, "系统中没有其他管理员持有全局 "+code+"，不能撤销自己的授权")
		}
	}

	return nil
}

// AdminLoginRecords 返回用户的论坛/后台登录记录。
func AdminLoginRecords(userID uint, page, pageSize int) (dto.Page[dto.LoginRecordItem], error) {
	if err := requireForumParticipant(userID); err != nil {
		return dto.Page[dto.LoginRecordItem]{}, err
	}

	page, pageSize = dao.Pagination(page, pageSize, 20, 100)

	db := store.DB.Model(&forummodels.LoginRecord{}).Where("user_id = ?", userID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dto.Page[dto.LoginRecordItem]{}, err
	}

	var records []forummodels.LoginRecord
	if err := db.Order("created_at DESC, id DESC").
		Offset(dao.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&records).Error; err != nil {
		return dto.Page[dto.LoginRecordItem]{}, err
	}

	items := make([]dto.LoginRecordItem, 0, len(records))
	for _, record := range records {
		items = append(items, dto.LoginRecordItem{
			ID:        record.ID,
			Channel:   string(record.Channel),
			IP:        record.IP,
			UserAgent: record.UserAgent,
			CreatedAt: dto.Timestamp(record.CreatedAt),
		})
	}

	return dto.NewPage(items, total, page, pageSize), nil
}

// AdminPermissionCatalog 返回权限点目录。
func AdminPermissionCatalog() []dto.PermissionItem {
	catalog := perms.All()
	items := make([]dto.PermissionItem, 0, len(catalog))
	for _, permission := range catalog {
		items = append(items, dto.PermissionItem{
			Code:        permission.Code,
			Group:       permission.Group,
			Scope:       string(permission.Scope),
			Description: permission.Description,
			Default:     permission.Default,
		})
	}
	return items
}

// AdminAuditLogs 返回审计日志。
func AdminAuditLogs(actorID uint, action, targetType string, targetID uint, from, to *time.Time, page, pageSize int) (dto.Page[dto.AuditLog], error) {
	db := store.DB.Model(&forummodels.AuditLog{})
	if actorID != 0 {
		db = db.Where("actor_id = ?", actorID)
	}
	if strings.TrimSpace(action) != "" {
		db = db.Where("action = ?", action)
	}
	if strings.TrimSpace(targetType) != "" {
		db = db.Where("target_type = ?", targetType)
	}
	if targetID != 0 {
		db = db.Where("target_id = ?", targetID)
	}
	if from != nil {
		db = db.Where("created_at >= ?", *from)
	}
	if to != nil {
		db = db.Where("created_at <= ?", *to)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dto.Page[dto.AuditLog]{}, err
	}

	page, pageSize = dao.Pagination(page, pageSize, 20, 100)

	var rows []forummodels.AuditLog
	if err := db.Order("created_at DESC, id DESC").
		Offset(dao.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return dto.Page[dto.AuditLog]{}, err
	}

	items := make([]dto.AuditLog, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.AuditLog{
			ID:         row.ID,
			ActorID:    row.ActorID,
			Action:     row.Action,
			TargetType: row.TargetType,
			TargetID:   row.TargetID,
			Detail:     []byte(row.Detail),
			IP:         row.IP,
			UserAgent:  row.UserAgent,
			CreatedAt:  dto.Timestamp(row.CreatedAt),
		})
	}

	return dto.NewPage(items, total, page, pageSize), nil
}

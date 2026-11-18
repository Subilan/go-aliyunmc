package perms

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	_ "embed"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"gorm.io/gorm"
)

//go:embed model.conf
var modelConf string

var (
	mu      sync.RWMutex
	current *casbin.Enforcer
	// reloadMu 串行化“读取数据库 + 构建 + 替换”，避免旧快照覆盖新快照。
	reloadMu sync.Mutex
)

// MustInitialize 从数据库加载用户权限节点并构建 enforcer。
func MustInitialize() {
	if err := Reload(); err != nil {
		log_util.Fatal("无法初始化 forum 权限管理器: %v", err)
	}
}

// Reload 重新读取 forum_user_permissions 并原子替换 enforcer。
// 失败时清空当前 enforcer（fail-closed），避免已撤销的权限继续生效。
func Reload() error {
	reloadMu.Lock()
	defer reloadMu.Unlock()

	var e *casbin.Enforcer
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		e, err = loadEnforcer()
		if err == nil {
			break
		}
		if attempt < 2 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err != nil {
		clearEnforcer()
		return fmt.Errorf("重新加载论坛权限失败: %w", err)
	}

	mu.Lock()
	current = e
	mu.Unlock()

	return nil
}

// loadEnforcer 读取数据库并构建新的 enforcer，不修改当前策略。
func loadEnforcer() (*casbin.Enforcer, error) {
	var rows []forummodels.UserPermission
	if err := store.DB.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("读取用户权限节点失败: %w", err)
	}

	policies := make([][]string, 0, len(rows))
	for _, row := range rows {
		if !Exists(row.PermissionCode) {
			log_util.Warn("忽略未知论坛权限点: permission=%s userId=%d", row.PermissionCode, row.UserID)
			continue
		}
		policies = append(policies, policyLine(row))
	}

	return newEnforcer(policies)
}

// clearEnforcer 在 reload 失败时清空策略，让非超级用户的权限判断一律拒绝。
func clearEnforcer() {
	mu.Lock()
	current = nil
	mu.Unlock()
}

func policyLine(row forummodels.UserPermission) []string {
	dom := "*"
	if row.ScopeBoardID != 0 {
		dom = "board:" + strconv.FormatUint(uint64(row.ScopeBoardID), 10)
	}
	return []string{subject(row.UserID), dom, row.PermissionCode}
}

func newEnforcer(policies [][]string) (*casbin.Enforcer, error) {
	m, err := casbinmodel.NewModelFromString(modelConf)
	if err != nil {
		return nil, fmt.Errorf("加载 forum Casbin 模型失败: %w", err)
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("创建 forum enforcer 失败: %w", err)
	}

	if len(policies) > 0 {
		if _, err := e.AddPolicies(policies); err != nil {
			return nil, fmt.Errorf("写入 forum 权限策略失败: %w", err)
		}
	}

	return e, nil
}

func subject(userID uint) string {
	return "user:" + strconv.FormatUint(uint64(userID), 10)
}

// Allow 判断用户是否拥有指定范围内（"*" 或 "board:<id>"）的权限点。
// 共享超级用户始终通过。
func Allow(user *models.User, scope, permission string) bool {
	if user == nil {
		return false
	}
	if user.Role == perm_util.RoleSuperuser {
		return true
	}
	return AllowID(user.ID, scope, permission)
}

// AllowID 判断用户 ID 是否拥有指定权限点，不做超级用户破窗。
func AllowID(userID uint, scope, permission string) bool {
	mu.RLock()
	e := current
	mu.RUnlock()

	if e == nil {
		return false
	}

	ok, err := e.Enforce(subject(userID), scope, permission)
	if err != nil {
		log_util.Warn("forum 权限判断失败: userId=%d scope=%s permission=%s error=%v", userID, scope, permission, err)
		return false
	}
	return ok
}

// IsSuperuser 判断是否为共享超级用户。
func IsSuperuser(user *models.User) bool {
	return user != nil && user.Role == perm_util.RoleSuperuser
}

// HasBoardPermission 判断用户在某个板块内是否拥有权限点（全局或板块级授权均可）。
// Casbin 模型本身已让板块请求匹配全局策略，因此只需一次判断。
func HasBoardPermission(user *models.User, boardID uint, permission string) bool {
	if user == nil {
		return false
	}
	return Allow(user, "board:"+strconv.FormatUint(uint64(boardID), 10), permission)
}

// Grant 是用户的一条权限节点记录。
type Grant struct {
	PermissionCode string `json:"permissionCode"`
	BoardID        *uint  `json:"boardId"`
}

// GrantsOf 返回用户的全部权限节点；超级用户返回全部权限点的全局授权。
func GrantsOf(user *models.User) []Grant {
	if user == nil {
		return []Grant{}
	}
	return GrantsForMany([]models.User{*user})[user.ID]
}

// GrantsForMany 批量返回用户的权限节点；超级用户返回全部权限点。
// 列表接口使用它避免逐用户查询。
func GrantsForMany(users []models.User) map[uint][]Grant {
	result := make(map[uint][]Grant, len(users))
	ids := make([]uint, 0, len(users))

	for _, user := range users {
		if IsSuperuser(&user) {
			grants := make([]Grant, 0, len(allCodes))
			for _, code := range allCodes {
				grants = append(grants, Grant{PermissionCode: code})
			}
			result[user.ID] = grants
			continue
		}
		result[user.ID] = []Grant{}
		ids = append(ids, user.ID)
	}

	if len(ids) == 0 {
		return result
	}

	rows, err := loadGrantRows(ids)
	if err != nil {
		log_util.Warn("读取用户权限节点失败: users=%v error=%v", ids, err)
		return result
	}

	for _, row := range rows {
		if !Exists(row.PermissionCode) {
			continue
		}
		grant := Grant{PermissionCode: row.PermissionCode}
		if row.ScopeBoardID != 0 {
			boardID := row.ScopeBoardID
			grant.BoardID = &boardID
		}
		result[row.UserID] = append(result[row.UserID], grant)
	}

	return result
}

// loadGrantRows 批量读取用户权限节点；调用方负责超级用户与未知权限点的处理。
func loadGrantRows(userIDs []uint) ([]forummodels.UserPermission, error) {
	var rows []forummodels.UserPermission
	err := store.DB.Where("user_id IN ?", userIDs).
		Order("user_id ASC, permission_code ASC, scope_board_id ASC").
		Find(&rows).Error
	return rows, err
}

// PermissionsOf 返回前台/后台 me 接口使用的全局与板块权限集合。
func PermissionsOf(user *models.User) (global []string, boards map[string][]string) {
	global = []string{}
	boards = make(map[string][]string)

	if user == nil {
		return global, boards
	}

	for _, grant := range GrantsForMany([]models.User{*user})[user.ID] {
		if grant.BoardID == nil {
			global = append(global, grant.PermissionCode)
			continue
		}
		key := strconv.FormatUint(uint64(*grant.BoardID), 10)
		boards[key] = append(boards[key], grant.PermissionCode)
	}

	return global, boards
}

// CanAccessAdmin 判断用户是否至少拥有一个 admin.access（全局或任一板块）。
func CanAccessAdmin(user *models.User) bool {
	return HasAnyGrant(user, "admin.access")
}

// BoardScopeFor 返回用户在指定权限点下可访问的板块范围。
// all=true 表示拥有全局授权（或超级用户），可访问全部板块。
func BoardScopeFor(user *models.User, permission string) (all bool, boards []uint) {
	if user == nil {
		return false, nil
	}
	if IsSuperuser(user) || AllowID(user.ID, "*", permission) {
		return true, nil
	}

	var rows []forummodels.UserPermission
	err := store.DB.Select("scope_board_id").
		Where("user_id = ? AND permission_code = ? AND scope_board_id <> 0", user.ID, permission).
		Find(&rows).Error
	if err != nil {
		log_util.Warn("读取后台板块授权失败: userId=%d error=%v", user.ID, err)
		return false, nil
	}

	seen := make(map[uint]bool)
	for _, row := range rows {
		if !seen[row.ScopeBoardID] {
			seen[row.ScopeBoardID] = true
			boards = append(boards, row.ScopeBoardID)
		}
	}
	sort.Slice(boards, func(i, j int) bool { return boards[i] < boards[j] })

	return false, boards
}

// HasAnyGrant 判断用户是否拥有指定权限点的全局或任意板块授权。
func HasAnyGrant(user *models.User, permission string) bool {
	all, boards := BoardScopeFor(user, permission)
	return all || len(boards) > 0
}

// ReplaceGrants 在事务内完整替换用户的权限节点，提交后调用方负责 Reload。
func ReplaceGrants(tx *gorm.DB, userID uint, grants []Grant, grantedBy uint) error {
	if err := tx.Where("user_id = ?", userID).Delete(&forummodels.UserPermission{}).Error; err != nil {
		return err
	}

	if len(grants) == 0 {
		return nil
	}

	rows := make([]forummodels.UserPermission, 0, len(grants))
	for _, grant := range grants {
		row := forummodels.UserPermission{
			UserID:         userID,
			PermissionCode: grant.PermissionCode,
			GrantedBy:      grantedBy,
		}
		if grant.BoardID != nil {
			row.ScopeBoardID = *grant.BoardID
		}
		rows = append(rows, row)
	}

	return tx.CreateInBatches(rows, 100).Error
}

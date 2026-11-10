package user_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/mid"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/perms"

	"github.com/gin-gonic/gin"
)

// UserPermissions 表示当前用户对各操作是否有权限。前端据此禁用按钮或显示提示。
type UserPermissions struct {
	// 用户账户
	CanChangePassword  bool `json:"can_change_password"`
	CanDeleteAccount   bool `json:"can_delete_account"`
	CanBindWhitelist   bool `json:"can_bind_whitelist"`
	CanUnbindWhitelist bool `json:"can_unbind_whitelist"`
	CanGetPreferences  bool `json:"can_get_preferences"`
	CanSetPreferences  bool `json:"can_set_preferences"`
	CanCreateUser      bool `json:"can_create_user"`
	CanChangeUserRole  bool `json:"can_change_user_role"`

	// 服务器
	CanQueryServer      bool `json:"can_query_server"`
	CanStopServer       bool `json:"can_stop_server"`
	CanAccessServerData bool `json:"can_access_server_data"`

	// 实例
	CanViewInstances        bool `json:"can_view_instances"`
	CanDeleteInstance       bool `json:"can_delete_instance"`
	CanCreateCustomInstance bool `json:"can_create_custom_instance"`

	// 任务
	CanViewTasks      bool `json:"can_view_tasks"`
	CanViewTaskOutput bool `json:"can_view_task_output"`
	CanTriggerTask    bool `json:"can_trigger_task"`
	CanRunBackup      bool `json:"can_run_backup"`
	CanRunArchive     bool `json:"can_run_archive"`

	// 状态监控
	CanWatchServerStatus   bool `json:"can_watch_server_status"`
	CanWatchInstanceStatus bool `json:"can_watch_instance_status"`

	// 采样数据
	CanViewPlayerHistory  bool `json:"can_view_player_history"`
	CanViewBalanceHistory bool `json:"can_view_balance_history"`

	// 其他
	CanViewAutoArchiveIdle   bool `json:"can_view_auto_archive_idle"`
	CanManageServerCountdown bool `json:"can_manage_server_countdown"`
}

// HandleGetPermissions 获取当前用户的权限信息
func HandleGetPermissions(c *gin.Context) (any, error) {
	role, exists := mid.CurrentRole(c)
	if !exists {
		return nil, nil
	}

	return UserPermissions{
		// 用户账户
		CanChangePassword:  checkEnforce(role, "/lab-v1/user/change-password", "POST"),
		CanDeleteAccount:   checkEnforce(role, "/lab-v1/user", "DELETE"),
		CanBindWhitelist:   checkEnforce(role, "/lab-v1/user/whitelist/bind", "POST"),
		CanUnbindWhitelist: checkEnforce(role, "/lab-v1/user/whitelist/unbind", "POST"),
		CanGetPreferences:  checkEnforce(role, "/lab-v1/user/preferences", "GET"),
		CanSetPreferences:  checkEnforce(role, "/lab-v1/user/preferences", "PUT"),
		CanCreateUser:      checkEnforce(role, "/lab-v1/user/create", "POST"),
		CanChangeUserRole:  checkEnforce(role, "/lab-v1/user/role", "POST"),

		// 服务器
		CanQueryServer:      checkEnforce(role, "/lab-v1/server/query", "GET"),
		CanStopServer:       checkEnforce(role, "/lab-v1/server/stop", "GET"),
		CanAccessServerData: checkEnforce(role, "/lab-v1/server/data", "GET"),

		// 实例
		CanViewInstances:        checkEnforce(role, "/lab-v1/instance/active", "GET"),
		CanDeleteInstance:       checkEnforce(role, "/lab-v1/instance/active", "DELETE"),
		CanCreateCustomInstance: checkCanExecute(role, "create-custom-instance"),

		// 任务
		CanViewTasks:      checkEnforce(role, "/lab-v1/task", "GET"),
		CanViewTaskOutput: checkEnforce(role, "/lab-v1/task/:id/output", "GET"),
		CanTriggerTask:    checkEnforce(role, "/lab-v1/task/trigger", "POST"),
		CanRunBackup:      role.Gte(perm_util.RoleOperator),
		CanRunArchive:     role.Gte(perm_util.RoleOperator),

		// 状态监控
		CanWatchServerStatus:   checkEnforce(role, "/lab-v1/state/snapshot/server-status", "GET"),
		CanWatchInstanceStatus: checkEnforce(role, "/lab-v1/state/snapshot/instance-status", "GET"),

		// 采样数据
		CanViewPlayerHistory:  checkEnforce(role, "/lab-v1/samples/player-list-history", "GET"),
		CanViewBalanceHistory: checkEnforce(role, "/lab-v1/samples/account-balance-history", "GET"),

		// 其他
		CanViewAutoArchiveIdle:   checkEnforce(role, "/lab-v1/monitor/auto-archive-idle", "GET"),
		CanManageServerCountdown: checkEnforce(role, "/lab-v1/monitor/auto-archive-idle/pause", "POST"),
	}, nil
}

func checkEnforce(role perm_util.Role, obj, act string) bool {
	ok, err := perms.Enforce(role, obj, act)
	if err != nil {
		return false
	}
	return ok
}

func checkCanExecute(role perm_util.Role, target string) bool {
	ok, err := perms.CanExecute(role, target)
	if err != nil {
		return false
	}
	return ok
}

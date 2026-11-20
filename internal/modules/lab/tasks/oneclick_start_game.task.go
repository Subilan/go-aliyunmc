package tasks

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/messages"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/global_states"
)

// oneClickStartGameTask 在一条任务记录内完成「创建实例、部署环境、启动服务器」。
// 触发前已由 checkOneClickStartGameTask 确认没有活跃实例，这里不会续跑已有实例。
func oneClickStartGameTask(tc *TaskContext, args map[string]any) error {
	if value, ok := args["startWhenCreated"].(bool); ok && !value {
		tc.println("[oneclick_start_game] 启动游戏需要启动实例，已忽略 startWhenCreated=false")
	}

	var params CreateInstanceTaskArgs
	if err := ShouldBindArgs(normalizeOneClickStartGameArgs(args), &params); err != nil {
		return err
	}

	// 触发校验与任务启动之间仍可能被其他部署链任务插入；此时 createInstance
	// 会以「已存在活跃实例」失败，而不会改动或续跑已有实例。
	tc.println("[oneclick_start_game] 阶段 1/3：创建实例")
	if err := createInstance(tc, params); err != nil {
		return fmt.Errorf("创建实例失败: %w", err)
	}

	tc.println("[oneclick_start_game] 阶段 2/3：部署环境")
	if err := deployTask(tc, nil); err != nil {
		return fmt.Errorf("部署失败: %w", err)
	}

	// 与单独触发 start_server 时保持一致：抢占式实例回收保护期间不启动服务器，
	// 即使启动成功也会因即将到来的强制断电而回档。
	if global_states.IsSpotInterruptionActive() {
		return fmt.Errorf("实例即将被回收，已跳过启动服务器")
	}

	tc.println("[oneclick_start_game] 阶段 3/3：启动服务器")
	if err := startServerTask(tc, nil); err != nil {
		return fmt.Errorf("启动服务器失败: %w", err)
	}

	tc.println("[oneclick_start_game] 启动游戏完成")
	notifyServerStarted()
	return nil
}

// checkOneClickStartGameTask 拒绝在已有活跃实例时触发一键启动。
// 已有实例的部署与启动应使用对应任务，避免一键启动悄悄续跑已有实例。
func checkOneClickStartGameTask(_ map[string]any) error {
	instance, err := dao.GetActiveInstanceDefaultNil()
	if err != nil {
		return err
	}

	if instance != nil {
		return h.HttpError(http.StatusConflict, "已存在活跃实例")
	}

	return nil
}

// serverStartedMessageData 是 server_started 消息模板可用的变量。
type serverStartedMessageData struct {
	Address string
	Port    uint16
}

// notifyServerStarted 在启动游戏成功后异步推送群消息。
// 消息模板来自 config/messages.toml；出错只记录日志，不影响任务结果。
func notifyServerStarted() {
	port := config.Get().Server.Port
	data := serverStartedMessageData{Port: port}
	if ip, err := dao.GetActiveInstanceIpNonEmpty(); err == nil {
		data.Address = net.JoinHostPort(ip, strconv.FormatUint(uint64(port), 10))
	}

	messages.Send(messages.ServerStarted, data)
}

// enforceOneClickStartGameTask 在创建新实例时沿用创建实例任务的权限规则。
// 已有实例时跳过创建规格的权限检查，让 checkOneClickStartGameTask 返回
// 「已存在实例」的提示，而不是先报权限不足。
func enforceOneClickStartGameTask(role perm_util.Role, args map[string]any) error {
	instance, err := dao.GetActiveInstanceDefaultNil()
	if err != nil {
		return err
	}
	if instance != nil {
		return nil
	}

	return enforceCreateInstanceTask(role, normalizeOneClickStartGameArgs(args))
}

// normalizeOneClickStartGameArgs 补全启动游戏的固定参数。
func normalizeOneClickStartGameArgs(args map[string]any) map[string]any {
	normalized := make(map[string]any, len(args)+2)
	for key, value := range args {
		normalized[key] = value
	}

	normalized["startWhenCreated"] = true

	if _, ok := normalized["useDefaultVSwitch"]; !ok {
		if vSwitchId, _ := normalized["vSwitchId"].(string); vSwitchId == "" {
			normalized["useDefaultVSwitch"] = true
		}
	}

	return normalized
}

package tasks

import (
	"fmt"

	"github.com/Subilan/go-aliyunmc/global_states"
	"github.com/Subilan/go-aliyunmc/perms"
	"github.com/Subilan/go-aliyunmc/store"
)

// oneClickStartGameTask 在一条任务记录内把服务器带到「已部署且在线」的终态。
//
// 它是幂等的：每次触发都会先读取当前状态，只执行尚未完成的阶段。
//   - 没有实例：创建实例 → 部署环境 → 启动服务器；
//   - 有实例但未部署：跳过创建，部署环境 → 启动服务器；
//   - 实例已部署：跳过创建与部署，只启动服务器；
//   - 服务器已在线：直接成功返回。
//
// 因此前端始终只需要一个「启动游戏」入口，失败后再次点击即可从失败点继续，
// 不需要根据实例状态切换到「部署」或「启动服务器」。
//
// 三段逻辑直接复用各自的任务函数：它们都通过 TaskContext 输出与上报步骤，
// 组合后仍然只有一条任务记录、一个 SSE 输出流。
func oneClickStartGameTask(tc *TaskContext, args map[string]any) error {
	if value, ok := args["startWhenCreated"].(bool); ok && !value {
		tc.println("[oneclick_start_game] 启动游戏需要启动实例，已忽略 startWhenCreated=false")
	}

	var params CreateInstanceTaskArgs
	if err := ShouldBindArgs(normalizeOneClickStartGameArgs(args), &params); err != nil {
		return err
	}

	instance, err := store.GetActiveInstanceDefaultNil()
	if err != nil {
		return err
	}
	instanceExisted := instance != nil

	if instance == nil {
		tc.println("[oneclick_start_game] 阶段 1/3：创建实例")
		if err := createInstance(tc, params); err != nil {
			return fmt.Errorf("创建实例失败: %w", err)
		}

		// 重新读取，拿到创建后的实例状态。
		instance, err = store.GetActiveInstance()
		if err != nil {
			return fmt.Errorf("读取实例信息失败: %w", err)
		}
	} else {
		tc.println("[oneclick_start_game] 检测到已有实例，跳过创建实例")
	}

	if !instance.IsDeployed {
		tc.println("[oneclick_start_game] 阶段 2/3：部署环境")
		if err := deployTask(tc, nil); err != nil {
			return fmt.Errorf("部署失败: %w", err)
		}
	} else {
		tc.println("[oneclick_start_game] 实例已完成部署，跳过部署环境")
	}

	// 只有本次触发之前实例就已存在时才参考服务器状态快照。刚创建的实例可能
	// 仍对应上一个实例留下的快照，不能据此判定服务器已经在线。
	if instanceExisted && SnapshotServerStatus != nil {
		if snap := SnapshotServerStatus(); snap.IsValid() && snap.Value.Online {
			tc.println("[oneclick_start_game] 服务器已在线，跳过启动")
			tc.println("[oneclick_start_game] 启动游戏完成")
			return nil
		}
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
	return nil
}

// enforceOneClickStartGameTask 与创建实例任务使用同一套权限规则：
// 普通用户只能使用默认交换机 + 最佳实例候选，自定义规格需要更高权限。
//
// 已有实例时启动游戏只做续跑，不会创建新实例，因此不再检查创建规格的权限。
func enforceOneClickStartGameTask(role perms.Role, args map[string]any) error {
	instance, err := store.GetActiveInstanceDefaultNil()
	if err != nil {
		return err
	}
	if instance != nil {
		return nil
	}

	return enforceCreateInstanceTask(role, normalizeOneClickStartGameArgs(args))
}

// normalizeOneClickStartGameArgs 补全启动游戏的固定参数：
//   - 启动游戏必须启动实例，后续的部署和启动步骤才能通过 SSH 执行；
//   - 未指定交换机时默认使用所在可用区的默认交换机，使调用方可以完全不带参数。
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

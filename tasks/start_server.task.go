package tasks

import (
	"context"
	"errors"
	"fmt"
	"github.com/Subilan/go-aliyunmc/global_states"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/remote_util"
	"github.com/Subilan/go-aliyunmc/store"
	"net/http"
	"time"

	"github.com/mcstatus-io/mcutil/v4/status"
)

func startServerTask(tc *TaskContext, _ map[string]any) error {
	tc.nextStep()
	ip, err := store.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return err
	}
	tc.println("尝试运行启动脚本")

	ctx, cancel := context.WithTimeout(tc.Context(), 5*time.Second)
	defer cancel()
	err = remote_util.ExecuteRemoteCommand(fmt.Sprintf(
		"cd /home/%s/server/archive && ./start.sh && sleep 0.5 && screen -S server -Q select . >/dev/null || exit 1",
		"mc",
	), ip, ctx, tc.println, false)

	if err != nil {
		return fmt.Errorf("启动脚本执行失败: %w", err)
	}

	cancel()

	tc.println("触发启动成功")
	tc.nextStep()
	tc.println("等待服务器就绪")

	ctx, cancel = context.WithTimeout(tc.Context(), 2*time.Minute)
	defer cancel()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

outer:
	for {
		select {
		case <-ctx.Done():
			tc.println("等待服务器就绪超时或被取消")
			return fmt.Errorf("等待服务器就绪超时或被取消")
		case <-ticker.C:
			dialCtx, dialCtxCancel := context.WithTimeout(tc.Context(), 5*time.Second)
			tc.println("尝试连接服务器...")
			resp, err := status.Modern(dialCtx, ip, 25565)
			dialCtxCancel()

			if err == nil {
				tc.println("成功连接到服务器")
				tc.println("服务器版本：" + resp.Version.Name.Raw)
				tc.println("延迟：" + resp.Latency.String())
				break outer
			}

			if errors.Is(err, context.Canceled) {
				tc.println("任务被取消")
				return fmt.Errorf("任务被取消")
			}
		}
	}

	tc.println("服务器启动成功")

	return nil
}

func checkStartServerTask(_ map[string]any) error {
	// 抢占式实例回收保护期间不允许启动服务器：实例即将被回收，即使启动成功也会因强制断电而回档。
	if global_states.IsSpotInterruptionActive() {
		return h.HttpError(http.StatusConflict, "实例即将被回收，服务器已被保护性停机，请等待实例恢复运行后再启动")
	}

	if err := checkMustHaveActiveDeployedRunningInstance(nil); err != nil {
		return err
	}

	snap := SnapshotServerStatus()

	if !snap.IsValid() {
		return h.HttpError(http.StatusServiceUnavailable, "无法获取最新的服务器状态")
	}

	if snap.Value.Online {
		return h.HttpError(http.StatusConflict, "服务器已在线")
	}

	return nil
}

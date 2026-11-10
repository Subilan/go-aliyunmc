package tasks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/global_states"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/remote_util"

	"github.com/mcstatus-io/mcutil/v4/response"
	"github.com/mcstatus-io/mcutil/v4/status"
)

const (
	minecraftArchiveDir      = "/home/mc/server/archive"
	minecraftLatestLogPath   = minecraftArchiveDir + "/logs/latest.log"
	maxMinecraftLogBytes     = 128 * 1024
	maxMinecraftLogLines     = 200
	serverInspectTimeout     = 5 * time.Second
	serverDiagnosticsTimeout = 10 * time.Second
)

type minecraftServerState struct {
	Screen    bool
	Java      bool
	PortInUse bool
	PortOwner string
	LogSize   int64
}

type startServerScriptVars struct {
	ArchiveDir string
}

type inspectServerScriptVars struct {
	Port    uint16
	LogPath string
}

type minecraftLogScriptVars struct {
	LogPath  string
	Offset   int64
	MaxBytes int
	MaxLines int
}

type lineCollector struct {
	mu    sync.Mutex
	lines []string
}

func (c *lineCollector) Add(line string) {
	c.mu.Lock()
	c.lines = append(c.lines, line)
	c.mu.Unlock()
}

func (c *lineCollector) Lines() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.lines...)
}

func (c *lineCollector) String() string {
	return strings.Join(c.Lines(), "\n")
}

func startServerTask(tc *TaskContext, _ map[string]any) error {
	tc.nextStep()
	ip, err := dao.GetActiveInstanceIpNonEmpty()
	if err != nil {
		return err
	}

	port := config.Get().Server.Port
	tc.println("检查服务器是否已经在线")

	probeCtx, probeCancel := context.WithTimeout(tc.Context(), 3*time.Second)
	resp, probeErr := status.Modern(probeCtx, ip, port)
	probeCancel()
	if probeErr == nil {
		printServerReady(tc, resp)
		tc.println("服务器已在线，跳过启动")
		return nil
	}
	if errors.Is(probeErr, context.Canceled) {
		tc.println("任务被取消")
		return fmt.Errorf("任务被取消")
	}

	inspectCtx, inspectCancel := context.WithTimeout(tc.Context(), serverInspectTimeout)
	state, err := inspectMinecraftServer(inspectCtx, ip, port)
	inspectCancel()
	if err != nil {
		return fmt.Errorf("检查服务器启动状态失败: %w", err)
	}

	if state.Java || state.Screen {
		tc.println("检测到服务器正在启动，跳过重复启动")
		return waitForMinecraftServer(tc, ip, port, state.LogSize)
	}
	if state.PortInUse {
		owner := state.PortOwner
		if owner == "" {
			owner = "无法读取占用进程信息"
		}
		return fmt.Errorf(
			"端口 %d 已被占用，且未检测到 Minecraft 启动进程\n端口占用信息: %s",
			port,
			owner,
		)
	}

	tc.println("尝试运行启动脚本")

	ctx, cancel := context.WithTimeout(tc.Context(), 5*time.Second)
	defer cancel()
	script, err := remote_util.RenderScriptTemplate(
		scriptTemplates,
		startServerTemplateName,
		startServerScriptVars{ArchiveDir: minecraftArchiveDir},
	)
	if err != nil {
		return fmt.Errorf("渲染启动脚本失败: %w", err)
	}
	err = remote_util.Mc(ip).ExecScript(ctx, script, tc.println)

	if err != nil {
		tc.println("启动脚本执行失败，正在读取 Minecraft 日志")
		return appendMinecraftLogDiagnostics(
			tc,
			ip,
			state.LogSize,
			fmt.Errorf("启动脚本执行失败: %w", err),
		)
	}

	cancel()

	tc.println("触发启动成功")
	return waitForMinecraftServer(tc, ip, port, state.LogSize)
}

func waitForMinecraftServer(tc *TaskContext, ip string, port uint16, logOffset int64) error {
	tc.nextStep()
	tc.println("等待服务器就绪")

	ctx, cancel := context.WithTimeout(tc.Context(), 2*time.Minute)
	defer cancel()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	waitStarted := time.Now()

outer:
	for {
		select {
		case <-ctx.Done():
			tc.println("等待服务器就绪超时或被取消")
			return appendMinecraftLogDiagnostics(
				tc,
				ip,
				logOffset,
				fmt.Errorf("等待服务器就绪超时或被取消"),
			)
		case <-ticker.C:
			dialCtx, dialCtxCancel := context.WithTimeout(tc.Context(), 5*time.Second)
			tc.println("尝试连接服务器...")
			resp, err := status.Modern(dialCtx, ip, port)
			dialCtxCancel()

			if err == nil {
				printServerReady(tc, resp)
				break outer
			}

			if errors.Is(err, context.Canceled) {
				tc.println("任务被取消")
				return fmt.Errorf("任务被取消")
			}

			if time.Since(waitStarted) < 5*time.Second {
				continue
			}

			inspectCtx, inspectCancel := context.WithTimeout(tc.Context(), serverInspectTimeout)
			state, inspectErr := inspectMinecraftServer(inspectCtx, ip, port)
			inspectCancel()
			if inspectErr != nil {
				tc.println("检查服务器进程失败，将继续等待：" + inspectErr.Error())
				continue
			}
			if state.Java || state.Screen {
				continue
			}

			detail := "Minecraft Java 进程已退出"
			if state.PortInUse {
				detail = "Minecraft Java 进程已退出，但端口仍被占用"
				if state.PortOwner != "" {
					detail += "：" + state.PortOwner
				}
			}
			tc.println("检测到服务器进程退出，正在读取 Minecraft 日志")
			return appendMinecraftLogDiagnostics(
				tc,
				ip,
				logOffset,
				fmt.Errorf("等待服务器就绪失败: %s", detail),
			)
		}
	}

	tc.println("服务器启动成功")

	return nil
}

func printServerReady(tc *TaskContext, resp *response.StatusModern) {
	tc.println("成功连接到服务器")
	tc.println("服务器版本：" + resp.Version.Name.Raw)
	tc.println("延迟：" + resp.Latency.String())
}

func inspectMinecraftServer(ctx context.Context, ip string, port uint16) (minecraftServerState, error) {
	var state minecraftServerState
	var collected lineCollector

	script, err := remote_util.RenderScriptTemplate(
		scriptTemplates,
		inspectServerTemplateName,
		inspectServerScriptVars{Port: port, LogPath: minecraftLatestLogPath},
	)
	if err != nil {
		return state, fmt.Errorf("渲染服务器状态检查脚本失败: %w", err)
	}
	err = remote_util.Mc(ip).ExecScript(ctx, script, collected.Add)
	if err != nil {
		return state, err
	}

	for _, line := range collected.Lines() {
		switch {
		case line == "screen=1":
			state.Screen = true
		case line == "java=1":
			state.Java = true
		case line == "port=1":
			state.PortInUse = true
		case strings.HasPrefix(line, "owner="):
			state.PortOwner = strings.TrimPrefix(line, "owner=")
		case strings.HasPrefix(line, "log_size="):
			if size, parseErr := strconv.ParseInt(strings.TrimPrefix(line, "log_size="), 10, 64); parseErr == nil {
				state.LogSize = size
			}
		}
	}
	return state, nil
}

func appendMinecraftLogDiagnostics(
	tc *TaskContext,
	ip string,
	offset int64,
	baseErr error,
) error {
	ctx, cancel := context.WithTimeout(tc.Context(), serverDiagnosticsTimeout)
	defer cancel()

	logTail, err := readMinecraftLogTail(ctx, ip, offset)
	if err != nil {
		return fmt.Errorf("%w\nMinecraft 日志读取失败: %v", baseErr, err)
	}
	if strings.TrimSpace(logTail) == "" {
		return fmt.Errorf("%w\nMinecraft latest.log: 无可用内容", baseErr)
	}
	return fmt.Errorf("%w\nMinecraft latest.log（本次启动）:\n%s", baseErr, logTail)
}

func readMinecraftLogTail(ctx context.Context, ip string, offset int64) (string, error) {
	if offset < 0 {
		offset = 0
	}

	script, err := remote_util.RenderScriptTemplate(
		scriptTemplates,
		minecraftLogTemplateName,
		minecraftLogScriptVars{
			LogPath:  minecraftLatestLogPath,
			Offset:   offset,
			MaxBytes: maxMinecraftLogBytes,
			MaxLines: maxMinecraftLogLines,
		},
	)
	if err != nil {
		return "", fmt.Errorf("渲染 Minecraft 日志读取脚本失败: %w", err)
	}

	var collected lineCollector
	err = remote_util.Mc(ip).ExecScript(ctx, script, collected.Add)
	if err != nil {
		return "", err
	}
	return collected.String(), nil
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

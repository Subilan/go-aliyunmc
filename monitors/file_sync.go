package monitors

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc/log_util"
	"go-aliyunmc/remote_util"
	"go-aliyunmc/store"
	"go-aliyunmc/tasks"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	waitSnapshotTimeout = 10 * time.Second
	syncFileTimeout     = 60 * time.Second
	syncDirTimeout      = 120 * time.Second
	initialJitterMax    = 3 * time.Second
)

// initialJitter 返回一个随机延迟，用于错开首次同步时的并发连接
func initialJitter() time.Duration {
	return time.Duration(rand.Int63n(int64(initialJitterMax)))
}

// FileSyncPoller 是一个定时轮询器，用于定期从远端服务器同步指定文件到本地缓存目录
type FileSyncPoller struct {
	stopChans map[string]chan struct{}
	mu        sync.Mutex
	logger    *log_util.NamedLogger
}

// newFileSyncPoller 创建一个新的 FileSyncPoller 实例
func newFileSyncPoller() *FileSyncPoller {
	return &FileSyncPoller{
		stopChans: make(map[string]chan struct{}),
		logger:    log_util.NewNamedLogger("[monitor/file-sync] ", "file-sync-monitor"),
	}
}

// run 为每个配置的文件和目录启动独立的轮询 goroutine，定期从远端服务器同步到本地缓存目录
func (p *FileSyncPoller) run(ctx context.Context) {
	// 确保本地缓存目录存在
	if err := p.ensureCacheDir(); err != nil {
		p.logger.Error("检查缓存目录失败: %v", err)
	}

	// 为每个文件创建独立的轮询 goroutine
	for _, fileConfig := range FileSyncC.Files {
		stopCh := make(chan struct{})
		p.mu.Lock()
		p.stopChans[fileConfig.RemotePath] = stopCh
		p.mu.Unlock()

		go p.pollFile(ctx, fileConfig, stopCh)
	}

	// 为每个目录创建独立的轮询 goroutine
	for _, dirConfig := range FileSyncC.Dirs {
		stopCh := make(chan struct{})
		p.mu.Lock()
		p.stopChans[dirConfig.RemotePath] = stopCh
		p.mu.Unlock()

		go p.pollDir(ctx, dirConfig, stopCh)
	}
}

// pollFile 为单个文件执行定时轮询
func (p *FileSyncPoller) pollFile(ctx context.Context, fileConfig FileConfig, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(fileConfig.PollIntervalSec) * time.Second)
	defer ticker.Stop()

	time.Sleep(initialJitter())
	p.syncFile(ctx, fileConfig)

	for {
		select {
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		case <-ticker.C:
			p.syncFile(ctx, fileConfig)
		}
	}
}

// syncTarget 执行同步前置检查，返回实例 IP；不满足条件时返回空字符串。
func (p *FileSyncPoller) syncTarget(name string) string {
	if tasks.IsArchiveTaskRunning() {
		p.logger.Info("归档流程正在执行，跳过%s同步", name)
		return ""
	}
	instance, err := store.GetActiveInstance()
	if err != nil {
		p.logger.Info("无法获取实例，跳过%s同步", name)
		return ""
	}
	if !instance.IsDeployed {
		p.logger.Info("实例未部署，跳过%s同步", name)
		return ""
	}
	snap, err := instanceMonitor.WaitSnapshot(waitSnapshotTimeout)
	if err != nil || !snap.IsValid() || snap.Value != "Running" {
		p.logger.Info("实例未运行或无法获取状态，跳过%s同步", name)
		return ""
	}
	return instance.Ip
}

// syncFile 执行单次文件同步
func (p *FileSyncPoller) syncFile(ctx context.Context, fileConfig FileConfig) {
	ip := p.syncTarget("文件")
	if ip == "" {
		return
	}

	localFullPath := filepath.Join(FileSyncC.LocalCacheRoot, fileConfig.LocalPath)
	if err := os.MkdirAll(filepath.Dir(localFullPath), 0755); err != nil {
		p.logger.Error("无法创建本地目录: %v", err)
		return
	}

	syncCtx, cancel := context.WithTimeout(ctx, syncFileTimeout)
	defer cancel()

	p.logger.Info("开始同步文件 %s", fileConfig.RemotePath)
	if err := remote_util.RsyncDownloadFile(syncCtx, fileConfig.RemotePath, localFullPath, ip); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			p.logger.Error("文件同步超时(%s)", syncFileTimeout)
			return
		}
		if errors.Is(err, context.Canceled) {
			p.logger.Error("文件同步取消")
			return
		}
		p.logger.Error("文件同步失败: %v", err)
		return
	}
	p.logger.Info("文件同步完成: %s -> %s", fileConfig.RemotePath, localFullPath)
}

// pollDir 为单个目录执行定时轮询
func (p *FileSyncPoller) pollDir(ctx context.Context, dirConfig DirConfig, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(dirConfig.PollIntervalSec) * time.Second)
	defer ticker.Stop()

	time.Sleep(initialJitter())
	p.syncDir(ctx, dirConfig)

	for {
		select {
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		case <-ticker.C:
			p.syncDir(ctx, dirConfig)
		}
	}
}

// syncDir 执行单次目录同步
func (p *FileSyncPoller) syncDir(ctx context.Context, dirConfig DirConfig) {
	ip := p.syncTarget("目录")
	if ip == "" {
		return
	}

	localFullPath := filepath.Join(FileSyncC.LocalCacheRoot, dirConfig.LocalPath)
	if err := os.MkdirAll(localFullPath, 0755); err != nil {
		p.logger.Error("无法创建本地目录: %v", err)
		return
	}

	syncCtx, cancel := context.WithTimeout(ctx, syncDirTimeout)
	defer cancel()

	p.logger.Info("开始同步目录 %s", dirConfig.RemotePath)
	if err := remote_util.RsyncDownloadDir(syncCtx, dirConfig.RemotePath, localFullPath, ip); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			p.logger.Error("目录同步超时(%s)", syncDirTimeout)
			return
		}
		if errors.Is(err, context.Canceled) {
			p.logger.Error("目录同步取消")
			return
		}
		p.logger.Error("目录同步失败: %v", err)
		return
	}
	p.logger.Info("目录同步完成: %s -> %s", dirConfig.RemotePath, localFullPath)
}

// ensureCacheDir 确保本地缓存目录存在
func (p *FileSyncPoller) ensureCacheDir() error {
	return ensureDirectory(FileSyncC.LocalCacheRoot)
}

// ensureDirectory 确保目录存在，不存在则创建
func ensureDirectory(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("路径不是目录: %s", path)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	// 创建目录
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("创建目录失败 %s: %w", path, err)
	}

	return nil
}

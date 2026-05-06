package monitors

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc/aliyun"
	"go-aliyunmc/global_states"
	"go-aliyunmc/log_util"
	"go-aliyunmc/remote_util"
	"go-aliyunmc/store"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	waitSnapshotTimeout = 10 * time.Second
	syncFileTimeout     = 60 * time.Second
	syncDirTimeout      = 120 * time.Second
	initialJitterMax    = 3 * time.Second
)

// dialSSHWithRetry 带重试的SSH连接，避免启动时多个goroutine同时拨号导致服务端限流
func dialSSHWithRetry(ip string) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.Password(aliyun.C.Ecs.RootPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := remote_util.DialWithRetry(fmt.Sprintf("%s:22", ip), sshConfig, 3)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %w", err)
	}
	return client, nil
}

// initialJitter 返回一个随机延迟，用于错开首次同步时的并发连接
func initialJitter() time.Duration {
	return time.Duration(rand.Int63n(int64(initialJitterMax)))
}

// downloadRemoteFile 通过SSH连接远程主机，并使用SFTP协议下载指定文件到本地
func downloadRemoteFile(ctx context.Context, ip string, remotePath string, localPath string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("同步任务已取消: %w", err)
	}

	client, err := dialSSHWithRetry(ip)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %w", err)
	}
	defer file.Close()

	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("打开远端文件失败: %w", err)
	}
	defer remoteFile.Close()

	copyErrCh := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(file, remoteFile)
		copyErrCh <- copyErr
	}()

	select {
	case <-ctx.Done():
		_ = client.Close()
		return fmt.Errorf("复制远端文件超时或取消: %w", ctx.Err())
	case copyErr := <-copyErrCh:
		if copyErr != nil {
			return fmt.Errorf("复制远端文件失败: %w", copyErr)
		}
	}

	return nil
}

// downloadRemoteDir 通过SSH连接远程主机，使用SFTP递归下载整个目录到本地
func downloadRemoteDir(ctx context.Context, ip string, remoteDir string, localDir string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("同步任务已取消: %w", err)
	}

	client, err := dialSSHWithRetry(ip)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	walker := sftpClient.Walk(remoteDir)
	for walker.Step() {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("同步任务已取消: %w", err)
		}

		stat := walker.Stat()
		if stat == nil {
			continue
		}

		if stat.IsDir() {
			continue
		}

		remotePath := walker.Path()
		relPath, err := filepath.Rel(remoteDir, remotePath)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}
		relPath = filepath.ToSlash(relPath)
		if strings.HasPrefix(relPath, "../") {
			continue
		}

		localFilePath := filepath.Join(localDir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
			return fmt.Errorf("创建本地目录失败 %s: %w", filepath.Dir(localFilePath), err)
		}

		remoteFile, err := sftpClient.Open(remotePath)
		if err != nil {
			return fmt.Errorf("打开远端文件失败 %s: %w", remotePath, err)
		}

		localFile, err := os.Create(localFilePath)
		if err != nil {
			remoteFile.Close()
			return fmt.Errorf("创建本地文件失败 %s: %w", localFilePath, err)
		}

		copyErrCh := make(chan error, 1)
		go func() {
			_, copyErr := io.Copy(localFile, remoteFile)
			localFile.Close()
			remoteFile.Close()
			copyErrCh <- copyErr
		}()

		select {
		case <-ctx.Done():
			return fmt.Errorf("复制远端文件超时或取消: %w", ctx.Err())
		case copyErr := <-copyErrCh:
			if copyErr != nil {
				return fmt.Errorf("复制远端文件失败 %s: %w", remotePath, copyErr)
			}
		}
	}

	if err := walker.Err(); err != nil {
		return fmt.Errorf("遍历远端目录失败: %w", err)
	}

	return nil
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

// syncFile 执行单次文件同步
func (p *FileSyncPoller) syncFile(ctx context.Context, fileConfig FileConfig) {
	if global_states.IsArchiving() {
		p.logger.Info("自动回收流程正在执行，跳过文件同步")
		return
	}
	// 获取实例
	instance, err := store.GetActiveInstance()

	if err != nil {
		p.logger.Info("无法获取实例，跳过")
		return
	}

	if !instance.IsDeployed {
		p.logger.Info("实例未部署，跳过")
		return
	}

	snap, err := instanceMonitor.WaitSnapshot(waitSnapshotTimeout)
	if err != nil || !snap.IsValid() || snap.Value != "Running" {
		p.logger.Info("实例未运行或无法获取状态，跳过")
		return
	}

	ip := instance.Ip

	// 构建本地目标文件地址
	localFullPath := filepath.Join(FileSyncC.LocalCacheRoot, fileConfig.LocalPath)

	// 创建文件地址所在的目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(localFullPath), 0755); err != nil {
		p.logger.Error("无法创建本地目录: %v", err)
		return
	}

	syncCtx, cancel := context.WithTimeout(ctx, syncFileTimeout)
	defer cancel()

	p.logger.Info("开始同步文件 %s", fileConfig.RemotePath)

	if err := downloadRemoteFile(syncCtx, ip, fileConfig.RemotePath, localFullPath); err != nil {
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
	if global_states.IsArchiving() {
		p.logger.Info("自动回收流程正在执行，跳过目录同步")
		return
	}

	instance, err := store.GetActiveInstance()
	if err != nil {
		p.logger.Info("无法获取实例，跳过目录同步")
		return
	}

	if !instance.IsDeployed {
		p.logger.Info("实例未部署，跳过目录同步")
		return
	}

	snap, err := instanceMonitor.WaitSnapshot(waitSnapshotTimeout)
	if err != nil || !snap.IsValid() || snap.Value != "Running" {
		p.logger.Info("实例未运行或无法获取状态，跳过目录同步")
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

	if err := downloadRemoteDir(syncCtx, instance.Ip, dirConfig.RemotePath, localFullPath); err != nil {
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

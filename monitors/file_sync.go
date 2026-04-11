package monitors

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc/aliyun"
	"go-aliyunmc/logs"
	"go-aliyunmc/states"
	"go-aliyunmc/store"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const syncFileTimeout = 30 * time.Second

// downloadRemoteFile 通过SSH连接远程主机，并使用SFTP协议下载指定文件到本地
func downloadRemoteFile(ctx context.Context, ip string, remotePath string, localPath string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("同步任务已取消: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.Password(aliyun.C.Ecs.RootPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), sshConfig)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
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

// FileSyncPoller 是一个定时轮询器，用于定期从远端服务器同步指定文件到本地缓存目录
type FileSyncPoller struct {
	stopChans map[string]chan struct{}
	mu        sync.Mutex
	logger    *logs.PrefixedLogger
}

// newFileSyncPoller 创建一个新的 FileSyncPoller 实例
func newFileSyncPoller() *FileSyncPoller {
	return &FileSyncPoller{
		stopChans: make(map[string]chan struct{}),
		logger:    logs.NewPrefixedLogger("[monitor/file-sync] "),
	}
}

// run 为每个配置的文件启动一个独立的轮询 goroutine，定期从远端服务器同步文件到本地缓存目录
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
}

// stop 停止所有文件同步轮询 goroutine
func (p *FileSyncPoller) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, stopCh := range p.stopChans {
		close(stopCh)
	}
}

// pollFile 为单个文件执行定时轮询
func (p *FileSyncPoller) pollFile(ctx context.Context, fileConfig FileConfig, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(fileConfig.PollIntervalSec) * time.Second)
	defer ticker.Stop()

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
	if states.IsArchiveTaskRunning() {
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

	if !states.StableSnapshotIsInstanceRunning(syncFileTimeout) {
		p.logger.Info("实例未运行，跳过")
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

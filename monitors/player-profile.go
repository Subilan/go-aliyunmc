package monitors

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/filelog"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// PlayerProfile 定期从服务器上通过 scp 下载玩家数据文件
func PlayerProfile(quit chan bool) {
	cfg := config.Cfg.Monitor.PlayerProfile
	var interval = cfg.IntervalDuration()
	var timeout = cfg.TimeoutDuration()

	logger := filelog.NewLogger("player-profile", "PlayerProfile")

	logger.Println("starting...")

	// 确保本地目录存在
	if err := os.MkdirAll(cfg.EssentialsDir, 0755); err != nil {
		logger.Println("error creating essentials dir:", err)
		return
	}
	if err := os.MkdirAll(cfg.StatsDir, 0755); err != nil {
		logger.Println("error creating stats dir:", err)
		return
	}

	ticker := time.NewTicker(interval)

	for {
		func() {
			logger.Println("running new player profile download task")

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			activeInstance, err := store.GetDeployedActiveInstance()

			if err != nil {
				logger.Println("no instance found, retry in", interval)
				return
			}

			// 下载 Essentials 玩家数据
			logger.Println("downloading Essentials player data...")
			err = downloadFiles(ctx, *activeInstance.Ip, cfg.EssentialsPath, cfg.EssentialsDir)

			if err != nil {
				logger.Println("error downloading essentials:", err)
			} else {
				logger.Println("essentials download ok")
			}

			// 下载 Minecraft 统计数据
			logger.Println("downloading Minecraft stats...")
			err = downloadFiles(ctx, *activeInstance.Ip, cfg.StatsPath, cfg.StatsDir)

			if err != nil {
				logger.Println("error downloading stats:", err)
			} else {
				logger.Println("stats download ok")
			}

			logger.Println("next download in", interval.String())
		}()

		select {
		case <-ticker.C:
			continue

		case <-quit:
			return
		}
	}
}

// downloadFiles 通过 SFTP 从服务器下载文件到本地目录
func downloadFiles(ctx context.Context, ip string, remotePath string, localDir string) error {
	// SSH 配置，使用与 helpers/remote/remote.go 相同的认证方式
	cfg := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Cfg.Aliyun.Ecs.RootPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// 建立 SSH 连接
	client, err := ssh.Dial("tcp", ip+":22", cfg)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	// 创建 SFTP 客户端
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("sftp new client: %w", err)
	}
	defer sftpClient.Close()

	// 查找远程文件
	remoteFiles, err := sftpClient.ReadDir(remotePath)
	if err != nil {
		return fmt.Errorf("read remote dir: %w", err)
	}

	// 下载每个文件
	for _, file := range remoteFiles {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return fmt.Errorf("download cancelled: %w", ctx.Err())
		default:
		}

		if file.IsDir() {
			continue
		}

		remoteFile := filepath.Join(remotePath, file.Name())
		localFile := filepath.Join(localDir, file.Name())

		err := downloadFile(ctx, sftpClient, remoteFile, localFile)
		if err != nil {
			return fmt.Errorf("download file %s: %w", remoteFile, err)
		}
	}

	return nil
}

// downloadFile 下载单个文件
func downloadFile(ctx context.Context, sftpClient *sftp.Client, remotePath string, localPath string) error {
	// 打开远程文件
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote file: %w", err)
	}
	defer remoteFile.Close()

	// 创建本地文件
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %w", err)
	}
	defer localFile.Close()

	// 使用 io.CopyBuffer 配合上下文检查，实现可中断的复制
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("download cancelled: %w", ctx.Err())
		default:
		}

		n, err := remoteFile.Read(buf)
		if n > 0 {
			_, writeErr := localFile.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("write local file: %w", writeErr)
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read remote file: %w", err)
		}
	}
}

package remote_util

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"go-aliyunmc/aliyun"
	"golang.org/x/crypto/ssh"
)

// UploadFileRemote 通过SFTP将本地文件上传到远程服务器。root参数决定使用root还是mc用户连接。
func UploadFileRemote(localPath, remotePath, ip string, ctx context.Context, root bool) error {
	user := "mc"
	password := aliyun.C.Ecs.ProdPassword
	if root {
		user = "root"
		password = aliyun.C.Ecs.RootPassword
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := DialWithRetry(fmt.Sprintf("%s:22", ip), sshConfig, 3)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	if err := sftpClient.MkdirAll(filepath.Dir(remotePath)); err != nil {
		return fmt.Errorf("创建远程目录失败: %w", err)
	}

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer localFile.Close()

	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	copyErrCh := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(remoteFile, localFile)
		copyErrCh <- copyErr
	}()

	select {
	case <-ctx.Done():
		_ = client.Close()
		return fmt.Errorf("上传文件超时或取消: %w", ctx.Err())
	case copyErr := <-copyErrCh:
		if copyErr != nil {
			return fmt.Errorf("上传文件失败: %w", copyErr)
		}
	}

	return nil
}

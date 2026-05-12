package remote_util

import (
	"context"
	"fmt"
	"os/exec"
)

const rsyncSSHOpts = "ssh -o StrictHostKeyChecking=no -o BatchMode=yes"

// RsyncUpload 使用rsync将本地文件上传到远程服务器（mc用户）
func RsyncUpload(ctx context.Context, localPath, remotePath, ip string) error {
	return runRsync(ctx, localPath, fmt.Sprintf("mc@%s:%s", ip, remotePath))
}

// RsyncDownloadFile 使用rsync从远程服务器下载单个文件（mc用户）
func RsyncDownloadFile(ctx context.Context, remotePath, localPath, ip string) error {
	return runRsync(ctx, fmt.Sprintf("mc@%s:%s", ip, remotePath), localPath)
}

// RsyncDownloadDir 使用rsync从远程服务器下载整个目录（mc用户）
func RsyncDownloadDir(ctx context.Context, remoteDir, localDir, ip string) error {
	return runRsync(ctx, fmt.Sprintf("mc@%s:%s/", ip, remoteDir), localDir+"/")
}

func runRsync(ctx context.Context, src, dst string) error {
	cmd := exec.CommandContext(ctx, "rsync", "-avz", "-e", rsyncSSHOpts, src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rsync失败: %w\n%s", err, string(out))
	}
	return nil
}

package remote_util

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"golang.org/x/crypto/ssh"
)

// Role 表示 SSH 登录所使用的账号。
type Role int

const (
	// RoleMc 是部署后用于日常操作的生产用户。
	RoleMc Role = iota
	// RoleRoot 是仅用于系统部署与维护的根用户。
	RoleRoot
)

func (r Role) username() string {
	if r == RoleRoot {
		return "root"
	}
	return "mc"
}

// authMethod 返回该角色的认证方式：mc 使用配置中的私钥，root 使用密码。
func (r Role) authMethod() (ssh.AuthMethod, error) {
	switch r {
	case RoleMc:
		if sshKey == nil {
			return nil, fmt.Errorf("未加载 %s 用户的SSH私钥", r.username())
		}
		return ssh.PublicKeys(sshKey), nil
	case RoleRoot:
		if config.Get().Aliyun.Ecs.RootPassword == "" {
			return nil, fmt.Errorf("未配置 root 用户的密码")
		}
		return ssh.Password(config.Get().Aliyun.Ecs.RootPassword), nil
	}
	return nil, fmt.Errorf("未知的 SSH 用户角色: %d", int(r))
}

// SSH 端口固定为 22，连接超时固定为 sshConnectTimeout。
const (
	sshPort           = "22"
	sshConnectTimeout = 10 * time.Second
)

// 连接建立的重试次数。
const (
	// maxConnectAttempts 是一次连接建立的最大尝试次数
	maxConnectAttempts = 3
	// probeConnectAttempts 是连通性探测的尝试次数
	probeConnectAttempts = 1
)

// Target 描述一台可通过 SSH 操作的远端主机，是所有远端操作的入口。
type Target struct {
	IP   string
	Role Role
}

// Mc 返回以生产用户(mc)登录的 Target。
func Mc(ip string) Target {
	return Target{IP: ip, Role: RoleMc}
}

// Root 返回以根用户(root)登录的 Target。
func Root(ip string) Target {
	return Target{IP: ip, Role: RoleRoot}
}

func (t Target) addr() string {
	return net.JoinHostPort(t.IP, sshPort)
}

func (t Target) clientConfig() (*ssh.ClientConfig, error) {
	auth, err := t.Role.authMethod()
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User:            t.Role.username(),
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshConnectTimeout,
	}, nil
}

// Connect 建立一条 SSH 连接，调用方负责 Close。
func (t Target) Connect(ctx context.Context) (*Conn, error) {
	return t.connect(ctx, maxConnectAttempts)
}

func (t Target) connect(ctx context.Context, attempts int) (*Conn, error) {
	config, err := t.clientConfig()
	if err != nil {
		return nil, err
	}

	client, err := dialWithRetry(ctx, t.addr(), config, attempts)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败(%s): %w", t.IP, err)
	}

	return &Conn{target: t, config: config, client: client}, nil
}

// Probe 建立一次 SSH 连接以验证连通性，不执行任何命令。
func (t Target) Probe(ctx context.Context) error {
	conn, err := t.connect(ctx, probeConnectAttempts)
	if err != nil {
		return err
	}

	return conn.Close()
}

// Exec 在远端执行一条命令，并按行回调输出。
func (t Target) Exec(ctx context.Context, cmd string, onLine func(string)) error {
	conn, err := t.Connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Exec(ctx, cmd, onLine)
}

// ExecScript 将脚本内容交给远端 bash 执行，并按行回调输出。
func (t Target) ExecScript(ctx context.Context, script string, onLine func(string)) error {
	conn, err := t.Connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.ExecScript(ctx, script, onLine)
}

// Upload 将本地文件上传到远端。
func (t Target) Upload(ctx context.Context, localPath, remotePath string, onLine func(string)) error {
	endpoint, err := t.transferEndpoint(remotePath)
	if err != nil {
		return err
	}
	return runRsync(ctx, config.Get().SSH.PrivateKeyPath, localPath, endpoint, onLine)
}

// DownloadFile 将远端文件下载到本地。
func (t Target) DownloadFile(ctx context.Context, remotePath, localPath string, onLine func(string)) error {
	endpoint, err := t.transferEndpoint(remotePath)
	if err != nil {
		return err
	}
	return runRsync(ctx, config.Get().SSH.PrivateKeyPath, endpoint, localPath, onLine)
}

// DownloadDir 将远端目录同步到本地，远端与本地路径都按目录语义处理。
func (t Target) DownloadDir(ctx context.Context, remoteDir, localDir string, onLine func(string)) error {
	endpoint, err := t.transferEndpoint(remoteDir + "/")
	if err != nil {
		return err
	}
	return runRsync(ctx, config.Get().SSH.PrivateKeyPath, endpoint, localDir+"/", onLine)
}

func (t Target) transferEndpoint(remotePath string) (string, error) {
	if t.Role != RoleMc {
		return "", fmt.Errorf("文件传输仅支持 mc 用户，当前为 %s", t.Role.username())
	}
	return fmt.Sprintf("mc@%s:%s", t.IP, remotePath), nil
}

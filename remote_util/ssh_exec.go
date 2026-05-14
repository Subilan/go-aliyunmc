package remote_util

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Subilan/go-aliyunmc/aliyun"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"golang.org/x/crypto/ssh"
)

// TryDialRoot 尝试使用root用户通过SSH连接远程主机，以验证连接是否成功
//   - host: 远程主机的IP地址或域名
//   - timeout: 连接超时时间，单位为秒
func TryDialRoot(host string, timeout time.Duration) bool {
	cfg := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(aliyun.C.Ecs.RootPassword),
		},
		Timeout:         timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // should not be used in prod but whatever...
	}

	client, err := DialWithRetry(host+":22", cfg, 3)
	if err != nil {
		return false
	}
	client.Close()
	return true
}

// RenderScriptTemplate 从指定路径读取脚本模板文件，并使用提供的变量进行渲染，返回渲染后的脚本内容
func RenderScriptTemplate(templatePath string, vars any) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("读取模板文件失败(%s): %w", templatePath, err)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("解析模板失败(%s): %w", templatePath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("渲染模板失败(%s): %w", templatePath, err)
	}

	script := buf.String()
	if !strings.HasSuffix(script, "\n") {
		script += "\n"
	}

	return script, nil
}

// ExecuteRemoteWithStartFn 是对 ExecuteWithStartFn 的封装，提供了root用户连接的选项
func ExecuteRemoteWithStartFn(params SSHExecParams, startFn func(*ssh.Session) error, root bool) error {
	if root {
		params.Config.User = "root"
		params.Config.Password = aliyun.C.Ecs.RootPassword
	} else {
		params.Config.User = "mc"
		params.Config.Password = aliyun.C.Ecs.ProdPassword
	}

	return ExecuteWithStartFn(params, startFn)
}

// ExecuteScriptRemote 通过SSH连接远程机器执行本地脚本内容。scriptPath 是本地脚本的路径。
func ExecuteScriptRemote(scriptPath string, ip string, ctx context.Context, onLine func(string), root bool) error {
	return ExecuteRemoteWithStartFn(SSHExecParams{
		Ctx:    ctx,
		IP:     ip,
		OnLine: onLine,
	}, func(session *ssh.Session) error {
		session.Stdin = strings.NewReader(scriptPath)
		return session.Start("bash -s")
	}, root)
}

// ExecuteRemoteCommand 通过SSH连接远程机器执行指令。cmd 是要执行的指令字符串。
func ExecuteRemoteCommand(cmd string, ip string, ctx context.Context, onLine func(string), root bool) error {
	return ExecuteRemoteWithStartFn(SSHExecParams{
		Ctx:    ctx,
		IP:     ip,
		OnLine: onLine,
	}, func(session *ssh.Session) error {
		return session.Start(cmd)
	}, root)
}

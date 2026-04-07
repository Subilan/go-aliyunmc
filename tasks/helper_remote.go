package tasks

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"go-aliyunmc/aliyun"

	"golang.org/x/crypto/ssh"
)

// tryDialRoot 尝试使用root用户通过SSH连接远程主机，以验证连接是否成功
//   - host: 远程主机的IP地址或域名
//   - timeout: 连接超时时间，单位为秒
func tryDialRoot(host string, timeout time.Duration) bool {
	cfg := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(aliyun.C.Ecs.RootPassword),
		},
		Timeout:         timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // should not be used in prod but whatever...
	}

	client, err := ssh.Dial("tcp", host+":22", cfg)

	if err != nil {
		return false
	}

	client.Close()

	return true
}

// renderScriptTemplate 从指定路径读取脚本模板文件，并使用提供的变量进行渲染，返回渲染后的脚本内容
func renderScriptTemplate(templatePath string, vars any) (string, error) {
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

// executeRemoteWithStartFn 通过SSH执行命令并处理输出
//   - ctx: 上下文对象，用于控制超时和取消
//   - ip: 远程机器的IP地址
//   - cfg: SSH连接配置，包括连接超时时间等
//   - script: 要执行的脚本内容
//   - startFn: 用于启动SSH会话的函数，参数为SSH会话对象，返回error
//   - onLine: 每当有新的输出行时调用的回调函数，参数为输出内容
func executeRemoteWithStartFn(ctx context.Context, ip string, cfg TaskSSHConfig, startFn func(*ssh.Session) error, onLine func(string), root bool) error {
	sshConfig := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(cfg.ConnectTimeoutSec) * time.Second,
	}

	if root {
		sshConfig.User = "root"
		sshConfig.Auth = []ssh.AuthMethod{ssh.Password(aliyun.C.Ecs.RootPassword)}
	} else {
		sshConfig.User = "mc"
		sshConfig.Auth = []ssh.AuthMethod{ssh.Password(aliyun.C.Ecs.ProdPassword)}
	}

	// 建立SSH连接
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, 22), sshConfig)
	if err != nil {
		return fmt.Errorf("SSH连接失败(%s): %w", ip, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取SSH标准输出失败: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取SSH错误输出失败: %w", err)
	}

	// 调用startFn来启动会话
	if err := startFn(session); err != nil {
		return fmt.Errorf("启动远程命令失败: %w", err)
	}

	if onLine == nil {
		onLine = func(string) {}
	}

	// scanPipe 是一个辅助函数，用于扫描给定的io.Reader（可以是stdout或stderr），
	// 并将每行输出通过onLine回调返回
	scanPipe := func(r io.Reader) error {
		scanner := bufio.NewScanner(r)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			onLine(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return nil
	}

	// wg 用于等待两个goroutine完成，scanErrCh 用于收集扫描输出时的错误
	var wg sync.WaitGroup
	scanErrCh := make(chan error, 2)

	wg.Add(2)
	// 扫描标准输出内容
	go func() {
		defer wg.Done()
		if err := scanPipe(stdout); err != nil {
			scanErrCh <- fmt.Errorf("读取SSH标准输出失败: %w", err)
		}
	}()
	// 扫描错误输出内容
	go func() {
		defer wg.Done()
		if err := scanPipe(stderr); err != nil {
			scanErrCh <- fmt.Errorf("读取SSH错误输出失败: %w", err)
		}
	}()

	// waitDone 用于等待远程命令执行完成
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- session.Wait()
	}()

	select {
	case waitErr := <-waitDone:
		// 当远程命令执行完成时，等待扫描输出的两个goroutine完成，并关闭错误通道
		wg.Wait()
		close(scanErrCh)

		for scanErr := range scanErrCh {
			if scanErr != nil {
				return scanErr
			}
		}

		if waitErr != nil {
			return fmt.Errorf("远程命令执行失败: %w", waitErr)
		}

		return nil

	case <-ctx.Done():
		// 如果上下文被取消，中止远程命令执行，并等待相关资源清理后返回上下文错误
		_ = session.Close()
		wg.Wait()
		return ctx.Err()
	}
}

// executeRemoteScript 通过SSH连接远程机器执行脚本内容。
func executeRemoteScript(ctx context.Context, ip string, cfg TaskSSHConfig, script string, onLine func(string), root bool) error {
	return executeRemoteWithStartFn(ctx, ip, cfg, func(session *ssh.Session) error {
		session.Stdin = strings.NewReader(script)
		return session.Start("bash -s")
	}, onLine, root)
}

// executeRemoteScriptPath 通过SSH连接远程机器执行已存在的脚本文件。
func executeRemoteScriptPath(ctx context.Context, ip string, cfg TaskSSHConfig, scriptPath string, onLine func(string), root bool) error {
	return executeRemoteWithStartFn(ctx, ip, cfg, func(session *ssh.Session) error {
		return session.Start(scriptPath)
	}, onLine, root)
}

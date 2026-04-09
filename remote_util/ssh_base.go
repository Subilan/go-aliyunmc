package remote_util

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHExecConfig struct {
	// User 是SSH连接的用户名。
	User              string
	// Password 是SSH连接的密码。
	Password          string
	// ConnectTimeoutSec 选填，是SSH连接的超时时间，单位为秒，默认为 10。
	ConnectTimeoutSec int
}

type SSHExecParams struct {
	// Ctx 用于控制SSH命令执行的上下文。
	Ctx context.Context
	// IP 是远程主机的IP地址或域名。
	IP string
	// Port 选填，是SSH连接的端口，默认为22。
	Port int
	// Config 定义了SSH连接的配置参数，包括用户名、密码和连接超时时间等信息。
	Config SSHExecConfig
	// OnLine 选填，是一个可选的回调函数，用于接收远程命令的输出行。每当远程命令输出一行文本时，SSHExec会调用这个函数并传递该行文本作为参数。
	OnLine func(string)
}

// ExecuteWithStartFn 建立一次SSH会话，调用 startFn 启动远端命令并逐行回传输出。
func ExecuteWithStartFn(p SSHExecParams, startFn func(*ssh.Session) error) error {
	if p.Config.ConnectTimeoutSec == 0 {
		p.Config.ConnectTimeoutSec = 10
	}

	if p.Port == 0 {
		p.Port = 22
	}

	if p.OnLine == nil {
		p.OnLine = func(string) {}
	}

	sshConfig := &ssh.ClientConfig{
		User:            p.Config.User,
		Auth:            []ssh.AuthMethod{ssh.Password(p.Config.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(p.Config.ConnectTimeoutSec) * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", p.IP, p.Port), sshConfig)
	if err != nil {
		return fmt.Errorf("SSH连接失败(%s): %w", p.IP, err)
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

	if err := startFn(session); err != nil {
		return fmt.Errorf("启动远程命令失败: %w", err)
	}

	scanPipe := func(r io.Reader) error {
		scanner := bufio.NewScanner(r)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			p.OnLine(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return nil
	}

	var wg sync.WaitGroup
	scanErrCh := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := scanPipe(stdout); err != nil {
			scanErrCh <- fmt.Errorf("读取SSH标准输出失败: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := scanPipe(stderr); err != nil {
			scanErrCh <- fmt.Errorf("读取SSH错误输出失败: %w", err)
		}
	}()

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- session.Wait()
	}()

	select {
	case waitErr := <-waitDone:
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

	case <-p.Ctx.Done():
		_ = session.Close()
		wg.Wait()
		return p.Ctx.Err()
	}
}

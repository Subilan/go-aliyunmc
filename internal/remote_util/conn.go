package remote_util

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Conn 是一条已建立的 SSH 连接，可被多个 goroutine 并发使用。
type Conn struct {
	target Target
	config *ssh.ClientConfig

	mu     sync.Mutex
	client *ssh.Client
}

// Target 返回这条连接所属的 Target。
func (c *Conn) Target() Target {
	return c.target
}

// Close 关闭连接。对已经失效的连接重复调用是安全的。
func (c *Conn) Close() error {
	c.mu.Lock()
	client := c.client
	c.client = nil
	c.mu.Unlock()

	if client == nil {
		return nil
	}
	return client.Close()
}

// Exec 在远端执行一条命令，并按行回调输出。
func (c *Conn) Exec(ctx context.Context, cmd string, onLine func(string)) error {
	return c.run(ctx, onLine, func(session *ssh.Session) error {
		return session.Start(cmd)
	})
}

// ExecScript 将脚本内容交给远端 bash 执行，并按行回调输出。
func (c *Conn) ExecScript(ctx context.Context, script string, onLine func(string)) error {
	return c.run(ctx, onLine, func(session *ssh.Session) error {
		session.Stdin = strings.NewReader(script)
		return session.Start("bash -s")
	})
}

// run 执行一次会话；失败说明连接本身已不可用，则丢弃当前连接，下一次操作会重新拨号。
func (c *Conn) run(ctx context.Context, onLine func(string), startFn func(*ssh.Session) error) error {
	err := c.runSession(ctx, onLine, startFn)
	if brokenConnection(err) {
		_ = c.Close()
	}
	return err
}

func (c *Conn) runSession(ctx context.Context, onLine func(string), startFn func(*ssh.Session) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if onLine == nil {
		onLine = func(string) {}
	}

	client, err := c.clientOrDial(ctx)
	if err != nil {
		return err
	}

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
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			onLine(scanner.Text())
		}
		return scanner.Err()
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

	case <-ctx.Done():
		_ = session.Close()
		wg.Wait()
		return ctx.Err()
	}
}

// clientOrDial 返回底层连接；若连接尚未建立或已被丢弃，则重新拨号。
func (c *Conn) clientOrDial(ctx context.Context) (*ssh.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		return c.client, nil
	}

	client, err := dialWithRetry(ctx, c.target.addr(), c.config, maxConnectAttempts)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败(%s): %w", c.target.IP, err)
	}
	c.client = client
	return client, nil
}

// brokenConnection 报告错误是否说明底层连接已经不可用。远端命令返回非零退出、调用方的 ctx 取消，两者都不算。
func brokenConnection(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var exitErr *ssh.ExitError
	return !errors.As(err, &exitErr)
}

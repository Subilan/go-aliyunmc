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

const (
	maxCommandOutputLines = 200
	maxCommandStdoutBytes = 32 * 1024
	maxCommandStderrBytes = 64 * 1024
	maxScannerTokenBytes  = 1024 * 1024
)

// CommandError 保留远端命令退出的实际输出。Error 包含诊断信息，Unwrap
// 返回底层错误，调用方可通过 errors.Is/errors.As 判断错误类型。
type CommandError struct {
	Err      error
	ExitCode int
	Stdout   string
	Stderr   string
}

func (e *CommandError) Error() string {
	var b strings.Builder
	b.WriteString("远程命令执行失败")
	if e.ExitCode >= 0 {
		fmt.Fprintf(&b, "（退出码 %d）", e.ExitCode)
	}
	if e.Err != nil {
		fmt.Fprintf(&b, ": %v", e.Err)
	}
	if e.Stdout != "" {
		b.WriteString("\nstdout:\n")
		b.WriteString(e.Stdout)
	}
	if e.Stderr != "" {
		b.WriteString("\nstderr:\n")
		b.WriteString(e.Stderr)
	}
	return b.String()
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

type outputTail struct {
	maxLines int
	maxBytes int
	lines    []string
	bytes    int
}

func (t *outputTail) Add(line string) {
	t.lines = append(t.lines, line)
	t.bytes += len(line) + 1

	for len(t.lines) > t.maxLines || t.bytes > t.maxBytes {
		if len(t.lines) == 0 {
			break
		}
		t.bytes -= len(t.lines[0]) + 1
		t.lines = t.lines[1:]
	}
}

func (t *outputTail) String() string {
	if len(t.lines) == 0 {
		return ""
	}

	var b strings.Builder
	for _, line := range t.lines {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	return b.String()
}

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

	scanPipe := func(r io.Reader, tail *outputTail) error {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), maxScannerTokenBytes)
		for scanner.Scan() {
			line := scanner.Text()
			onLine(line)
			tail.Add(line)
		}
		return scanner.Err()
	}

	stdoutTail := &outputTail{maxLines: maxCommandOutputLines, maxBytes: maxCommandStdoutBytes}
	stderrTail := &outputTail{maxLines: maxCommandOutputLines, maxBytes: maxCommandStderrBytes}

	var wg sync.WaitGroup
	scanErrCh := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := scanPipe(stdout, stdoutTail); err != nil {
			scanErrCh <- fmt.Errorf("读取SSH标准输出失败: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := scanPipe(stderr, stderrTail); err != nil {
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

		var scanErr error
		for err := range scanErrCh {
			if err != nil && scanErr == nil {
				scanErr = err
			}
		}

		if waitErr != nil || scanErr != nil {
			cause := waitErr
			if cause == nil {
				cause = scanErr
			}
			return &CommandError{
				Err:      cause,
				ExitCode: exitCode(waitErr),
				Stdout:   stdoutTail.String(),
				Stderr:   stderrTail.String(),
			}
		}

		return nil

	case <-ctx.Done():
		_ = session.Close()
		wg.Wait()
		return ctx.Err()
	}
}

func exitCode(err error) int {
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitStatus()
	}
	return -1
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

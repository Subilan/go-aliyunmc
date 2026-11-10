package remote_util

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// dialWithRetry 尝试建立 SSH 连接，最多 attempts 次（含首次），退避间隔为 2s、4s、8s……，
// 拨号与重试等待都会响应 ctx 取消。认证失败一类的永久性错误立刻返回。
func dialWithRetry(ctx context.Context, addr string, config *ssh.ClientConfig, attempts int) (*ssh.Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var lastErr error
	for i := range attempts {
		if i > 0 {
			if err := sleepCtx(ctx, time.Duration(1<<(i-1))*2*time.Second); err != nil {
				return nil, err
			}
		}

		client, err := dialOnce(ctx, addr, config)
		if err == nil {
			return client, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if permanentDialError(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func dialOnce(ctx context.Context, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	dialer := &net.Dialer{Timeout: config.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return ssh.NewClient(clientConn, chans, reqs), nil
}

// permanentDialError 报告该错误重试也不会成功：认证失败与算法协商失败取决于配置或两端环境。
// x/crypto 没有为认证失败导出哨兵错误，只能匹配它稳定的报文。
func permanentDialError(err error) bool {
	if err == nil {
		return false
	}

	var negotiation *ssh.AlgorithmNegotiationError
	if errors.As(err, &negotiation) {
		return true
	}

	return strings.Contains(err.Error(), "ssh: unable to authenticate")
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

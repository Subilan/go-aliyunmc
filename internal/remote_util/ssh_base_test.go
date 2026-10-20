package remote_util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestPermanentDialError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "认证失败不重试",
			err:  fmt.Errorf("ssh: handshake failed: %w", errors.New("ssh: unable to authenticate, attempted methods [none], no supported methods remain")),
			want: true,
		},
		{
			name: "算法协商失败不重试",
			err:  &ssh.AlgorithmNegotiationError{What: "host key"},
			want: true,
		},
		{
			name: "连接被拒绝可重试",
			err:  &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			want: false,
		},
		{
			name: "握手超时可重试",
			err:  fmt.Errorf("ssh: handshake failed: %w", context.DeadlineExceeded),
			want: false,
		},
		{
			name: "无错误",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := permanentDialError(tc.err); got != tc.want {
				t.Fatalf("permanentDialError() = %v, 期望 %v", got, tc.want)
			}
		})
	}
}

func TestBrokenConnection(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "远端命令返回非零退出不算断线",
			err:  fmt.Errorf("远程命令执行失败: %w", &ssh.ExitError{}),
			want: false,
		},
		{
			name: "调用方取消不算断线",
			err:  fmt.Errorf("远程命令执行失败: %w", context.Canceled),
			want: false,
		},
		{
			name: "调用方超时不算断线",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "对端关闭连接算断线",
			err:  fmt.Errorf("远程命令执行失败: %w", io.EOF),
			want: true,
		},
		{
			name: "连接已关闭算断线",
			err:  fmt.Errorf("创建SSH会话失败: %w", net.ErrClosed),
			want: true,
		},
		{
			name: "无错误",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := brokenConnection(tc.err); got != tc.want {
				t.Fatalf("brokenConnection() = %v, 期望 %v", got, tc.want)
			}
		})
	}
}

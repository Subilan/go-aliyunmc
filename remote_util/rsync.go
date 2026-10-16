package remote_util

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// rsyncSSHCommand 返回 rsync 的远端 shell 命令。身份由 -i 与 IdentitiesOnly=yes 显式指定，
// 避免回落到服务账号家目录或 agent 中的任意密钥。密钥路径由调用方（Target）传入，
// 因此本函数与传输层不依赖任何全局状态。
func rsyncSSHCommand(privateKeyPath string) string {
	return fmt.Sprintf(
		"ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o BatchMode=yes",
		strconv.Quote(privateKeyPath),
	)
}

// rsyncOutputWriter 将 rsync 的输出按行回调，并保留末尾若干行用于出错时的诊断信息。
type rsyncOutputWriter struct {
	mu        sync.Mutex
	pending   []byte
	tail      []string
	tailLimit int
	onLine    func(string)
}

func (w *rsyncOutputWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.pending = append(w.pending, p...)
	for {
		i := bytes.IndexByte(w.pending, '\n')
		if i < 0 {
			break
		}
		w.emit(string(w.pending[:i]))
		w.pending = w.pending[i+1:]
	}
	return len(p), nil
}

func (w *rsyncOutputWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.pending) > 0 {
		w.emit(string(w.pending))
		w.pending = nil
	}
}

func (w *rsyncOutputWriter) emit(line string) {
	w.tail = append(w.tail, line)
	if w.tailLimit > 0 && len(w.tail) > w.tailLimit {
		w.tail = w.tail[len(w.tail)-w.tailLimit:]
	}
	if w.onLine != nil {
		w.onLine(line)
	}
}

func (w *rsyncOutputWriter) detail() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	var buf bytes.Buffer
	for i, line := range w.tail {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
	}
	return buf.String()
}

func runRsync(ctx context.Context, privateKeyPath, src, dst string, onLine func(string)) error {
	if ctx == nil {
		ctx = context.Background()
	}

	writer := &rsyncOutputWriter{tailLimit: 20, onLine: onLine}

	cmd := exec.CommandContext(ctx, "rsync", "-avz", "-e", rsyncSSHCommand(privateKeyPath), src, dst)
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdout = writer
	cmd.Stderr = writer

	err := cmd.Run()
	writer.flush()

	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("rsync被中断: %w", ctxErr)
		}
		if detail := writer.detail(); detail != "" {
			return fmt.Errorf("rsync失败: %w\n%s", err, detail)
		}
		return fmt.Errorf("rsync失败: %w", err)
	}

	return nil
}

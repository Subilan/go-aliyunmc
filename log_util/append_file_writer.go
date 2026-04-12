package log_util

import (
	"os"
	"path/filepath"
	"sync"
)

// appendFileWriter 是一个io.Writer实现，用于将日志内容追加写入指定目录下的文件
type appendFileWriter struct {
	dir      string
	filename string
	mu       sync.Mutex
}

// Write 将日志内容追加写入文件，如果文件或目录不存在则会创建
func (w *appendFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return 0, err
	}

	path := filepath.Join(w.dir, w.filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return f.Write(p)
}

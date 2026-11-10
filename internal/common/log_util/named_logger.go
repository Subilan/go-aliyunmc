package log_util

import (
	"os"
	"path/filepath"
	"strings"
)

// NamedLogger 是按名称写入独立日志文件的 Logger，方法由 Logger 提供。
type NamedLogger struct {
	*Logger
}

// NewNamedLogger 创建一个新的 NamedLogger 实例
//   - prefix: 日志前缀，通常用于标识日志来源或模块
//   - name: 用于生成日志文件名称的基础名称，经过清洗后会用作日志文件的前缀
func NewNamedLogger(prefix string, name string) *NamedLogger {
	base := sanitizeBaseName(name)
	return &NamedLogger{newLogger(loggerOptions{
		prefix:     prefix,
		logPath:    filepath.Join(defaultLogsDir, base+"-latest.log"),
		maxSizeMB:  defaultRotateMaxMB,
		console:    os.Stdout,
		consoleErr: os.Stderr,
	})}
}

// sanitizeBaseName 将输入的字符串转换为适合用作日志文件基础名称的格式，替换或移除不合法字符
func sanitizeBaseName(baseName string) string {
	base := strings.TrimSpace(baseName)
	if base == "" {
		return "prefixed_"
	}

	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_", "[", "", "]", "", "(", "", ")", "")
	base = replacer.Replace(base)
	if base == "" {
		return "prefixed_"
	}

	return base
}

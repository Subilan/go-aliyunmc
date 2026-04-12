package log_util

import (
	"os"
	"strings"
)

// NamedLogger 是一个带有前缀的日志记录器，支持不同日志级别的输出，并将日志写入以指定名称为基础的文件中
type NamedLogger struct {
	prefix string
	d      *doubleLogger
	i      *doubleLogger
	w      *doubleLogger
	e      *doubleLogger
	f      *doubleLogger
}

// NewNamedLogger 创建一个新的 NamedLogger 实例
//  - prefix: 日志前缀，通常用于标识日志来源或模块
//  - name: 用于生成日志文件名称的基础名称，经过清洗后会用作日志文件的前缀
func NewNamedLogger(prefix string, name string) *NamedLogger {
	base := sanitizeBaseName(name)
	fileOut := &rotatingFileWriter{
		dir:      defaultLogsDir,
		baseName: base,
		maxBytes: defaultRotateMaxBytes,
	}

	return &NamedLogger{
		prefix: prefix,
		d:      newDoubleLogger(os.Stdout, fileOut, DevLevel, colorBlue),
		i:      newDoubleLogger(os.Stdout, fileOut, InfoLevel, colorGreen),
		w:      newDoubleLogger(os.Stdout, fileOut, WarnLevel, colorYellow),
		e:      newDoubleLogger(os.Stdout, fileOut, ErrorLevel, colorRed),
		f:      newDoubleLogger(os.Stderr, fileOut, FatalLevel, colorRed),
	}
}

func (l *NamedLogger) Dev(msg string, args ...any) {
	writef(l.d, l.prefix+msg, args...)
}

func (l *NamedLogger) Info(msg string, args ...any) {
	writef(l.i, l.prefix+msg, args...)
}

func (l *NamedLogger) Warn(msg string, args ...any) {
	writef(l.w, l.prefix+msg, args...)
}

func (l *NamedLogger) Error(msg string, args ...any) {
	writef(l.e, l.prefix+msg, args...)
}

func (l *NamedLogger) Fatal(msg string, args ...any) {
	writef(l.f, l.prefix+msg, args...)
	os.Exit(1)
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
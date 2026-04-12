package log_util

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	DevLevel   = "[DEV] "
	InfoLevel  = "[INFO] "
	WarnLevel  = "[WARN] "
	ErrorLevel = "[ERROR] "
	FatalLevel = "[FATAL] "

	colorBlue   = "\x1b[34m"
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
	colorRed    = "\x1b[31m"
	colorReset  = "\x1b[0m"
)

const defaultMainLogName = "main.log"

var defaultLogsDir = resolveDefaultLogsDir()

var mainFileOut = &appendFileWriter{dir: defaultLogsDir, filename: defaultMainLogName}

var d = newDoubleLogger(os.Stdout, mainFileOut, DevLevel, colorBlue)
var i = newDoubleLogger(os.Stdout, mainFileOut, InfoLevel, colorGreen)
var w = newDoubleLogger(os.Stdout, mainFileOut, WarnLevel, colorYellow)
var e = newDoubleLogger(os.Stdout, mainFileOut, ErrorLevel, colorRed)
var f = newDoubleLogger(os.Stderr, mainFileOut, FatalLevel, colorRed)

type logPrinter interface {
	Printf(format string, v ...any)
}

func newLogger(out io.Writer, prefix string) *log.Logger {
	return log.New(out, prefix, log.LstdFlags)
}

func writef(l logPrinter, msg string, args ...any) {
	l.Printf(msg, args...)
}

// Dev 输出调试日志
func Dev(msg string, args ...any) {
	writef(d, msg, args...)
}

// Info 输出信息日志
func Info(msg string, args ...any) {
	writef(i, msg, args...)
}

// Warn 输出警告日志
func Warn(msg string, args ...any) {
	writef(w, msg, args...)
}

// Error 输出错误日志
func Error(msg string, args ...any) {
	writef(e, msg, args...)
}

// Fatal 输出致命错误日志并退出程序
func Fatal(msg string, args ...any) {
	writef(f, msg, args...)
	os.Exit(1)
}

func uniqueArchivePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func resolveDefaultLogsDir() string {
	if dir := strings.TrimSpace(os.Getenv("GO_ALIYUNMC_LOG_DIR")); dir != "" {
		return dir
	}

	if strings.HasSuffix(os.Args[0], ".test") {
		return filepath.Join(os.TempDir(), "go-aliyunmc-logs")
	}

	return "logs"
}

func colorizePrefix(prefix string, color string) string {
	trimmed := strings.TrimRight(prefix, " ")
	spaces := prefix[len(trimmed):]
	if trimmed == "" {
		return prefix
	}
	return color + trimmed + colorReset + spaces
}

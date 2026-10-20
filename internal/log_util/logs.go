// Package log_util 基于 log/slog 提供统一的日志门面。
// 控制台输出带颜色，文件输出为纯文本并按大小轮转。
package log_util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Subilan/go-aliyunmc/internal/paths"

	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
	LevelFatal = slog.Level(12)

	timeFormat = "2006/01/02 15:04:05"

	defaultMainLogName = "main.log"

	// defaultRotateMaxMB 是具名日志文件的轮转阈值，单位为 MiB
	defaultRotateMaxMB = 1
	// mainRotateMaxMB 是主日志文件的轮转阈值，单位为 MiB
	mainRotateMaxMB = 100
)

var defaultLogsDir = resolveDefaultLogsDir()

var defaultLogger = newLogger(loggerOptions{
	logPath:    filepath.Join(defaultLogsDir, defaultMainLogName),
	maxSizeMB:  mainRotateMaxMB,
	console:    os.Stdout,
	consoleErr: os.Stderr,
})

// Logger 将同一条记录同时写入控制台与轮转文件。
// 除 Fatal 写入 stderr 外，其余级别均写入 stdout。
type Logger struct {
	prefix string
	out    *slog.Logger
	err    *slog.Logger
}

type loggerOptions struct {
	prefix     string
	logPath    string
	maxSizeMB  int
	console    io.Writer
	consoleErr io.Writer
	color      bool
}

func newLogger(o loggerOptions) *Logger {
	file := fileHandler(newRotatingWriter(o.logPath, o.maxSizeMB))
	return &Logger{
		prefix: o.prefix,
		out:    slog.New(newMultiHandler(consoleHandler(o.console, o.color), file)),
		err:    slog.New(newMultiHandler(consoleHandler(o.consoleErr, o.color), file)),
	}
}

// Debug 输出调试日志
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info 输出信息日志
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn 输出警告日志
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error 输出错误日志
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Fatal 输出致命错误日志并退出程序
func Fatal(msg string, args ...any) {
	defaultLogger.Fatal(msg, args...)
}

func (l *Logger) log(level slog.Level, msg string, args ...any) {
	logger := l.out
	if level >= LevelFatal {
		logger = l.err
	}
	logger.Log(context.Background(), level, l.prefix+formatMessage(msg, args...))
}

// Debug 输出调试日志
func (l *Logger) Debug(msg string, args ...any) {
	l.log(LevelDebug, msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.log(LevelInfo, msg, args...)
}

// Warn 输出警告日志
func (l *Logger) Warn(msg string, args ...any) {
	l.log(LevelWarn, msg, args...)
}

// Error 输出错误日志
func (l *Logger) Error(msg string, args ...any) {
	l.log(LevelError, msg, args...)
}

// Fatal 输出致命错误日志并退出程序
func (l *Logger) Fatal(msg string, args ...any) {
	l.log(LevelFatal, msg, args...)
	os.Exit(1)
}

func formatMessage(msg string, args ...any) string {
	return fmt.Sprintf(msg, args...)
}

func newRotatingWriter(path string, maxSizeMB int) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:  path,
		MaxSize:   maxSizeMB,
		LocalTime: true,
	}
}

func consoleHandler(w io.Writer, forceColor bool) slog.Handler {
	return tint.NewHandler(w, &tint.Options{
		Level:      LevelDebug,
		TimeFormat: timeFormat,
		NoColor:    !(forceColor || isTerminal(w)),
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if isFatalLevel(a) {
				return tint.Attr(9, slog.String(slog.LevelKey, "FATAL"))
			}
			return a
		},
	})
}

func fileHandler(w io.Writer) slog.Handler {
	return slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: LevelDebug,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if isFatalLevel(a) {
				return slog.String(slog.LevelKey, "FATAL")
			}
			return a
		},
	})
}

func isFatalLevel(a slog.Attr) bool {
	level, ok := a.Value.Any().(slog.Level)
	return a.Key == slog.LevelKey && ok && level >= LevelFatal
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

type multiHandler []slog.Handler

func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return multiHandler(handlers)
}

func (m multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var errs []error
	for _, h := range m {
		if err := h.Handle(ctx, record.Clone()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make(multiHandler, len(m))
	for i, h := range m {
		handlers[i] = h.WithAttrs(attrs)
	}
	return handlers
}

func (m multiHandler) WithGroup(name string) slog.Handler {
	handlers := make(multiHandler, len(m))
	for i, h := range m {
		handlers[i] = h.WithGroup(name)
	}
	return handlers
}

func resolveDefaultLogsDir() string {
	if strings.HasSuffix(os.Args[0], ".test") {
		return filepath.Join(os.TempDir(), "github.com/Subilan/go-aliyunmc-logs")
	}

	return paths.LogsDir()
}

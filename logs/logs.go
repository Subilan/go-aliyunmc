package logs

import (
	"log"
	"os"
)

type Logger interface {
	Dev(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Fatal(msg string, args ...any)
}

type PrefixedLogger struct {
	prefix string
}

var ldev = newLogger(os.Stdout, "[DEV] ")
var linfo = newLogger(os.Stdout, "[INFO] ")
var lwarn = newLogger(os.Stdout, "[WARN] ")
var lerror = newLogger(os.Stdout, "[ERROR] ")
var lfatal = newLogger(os.Stderr, "[FATAL] ")

func NewPrefixedLogger(prefix string) *PrefixedLogger {
	return &PrefixedLogger{prefix: prefix}
}

func (l *PrefixedLogger) Dev(msg string, args ...any) {
	writef(ldev, l.prefix+msg, args...)
}

func (l *PrefixedLogger) Info(msg string, args ...any) {
	writef(linfo, l.prefix+msg, args...)
}

func (l *PrefixedLogger) Warn(msg string, args ...any) {
	writef(lwarn, l.prefix+msg, args...)
}

func (l *PrefixedLogger) Error(msg string, args ...any) {
	writef(lerror, l.prefix+msg, args...)
}

func (l *PrefixedLogger) Fatal(msg string, args ...any) {
	writef(lfatal, l.prefix+msg, args...)
	os.Exit(1)
}

func newLogger(out *os.File, prefix string) *log.Logger {
	return log.New(out, prefix, log.LstdFlags)
}

func writef(l *log.Logger, msg string, args ...any) {
	l.Printf(msg, args...)
}

// Dev 输出调试日志
func Dev(msg string, args ...any) {
	writef(ldev, msg, args...)
}

// Info 输出信息日志
func Info(msg string, args ...any) {
	writef(linfo, msg, args...)
}

// Warn 输出警告日志
func Warn(msg string, args ...any) {
	writef(lwarn, msg, args...)
}

// Error 输出错误日志
func Error(msg string, args ...any) {
	writef(lerror, msg, args...)
}

// Fatal 输出致命错误日志并退出程序
func Fatal(msg string, args ...any) {
	writef(lfatal, msg, args...)
	os.Exit(1)
}

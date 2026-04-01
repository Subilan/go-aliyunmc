package logs

import (
	"log"
	"os"
)

func newLogger(out *os.File, prefix string) *log.Logger {
	return log.New(out, prefix, log.LstdFlags)
}

func writef(l *log.Logger, msg string, args ...any) {
	l.Printf(msg, args...)
}

var dev = newLogger(os.Stdout, "[DEV] ")
var info = newLogger(os.Stdout, "[INFO] ")
var warn = newLogger(os.Stdout, "[WARN] ")
var error = newLogger(os.Stdout, "[ERROR] ")
var fatal = newLogger(os.Stderr, "[FATAL] ")

// Dev 输出调试日志
func Dev(msg string, args ...any) {
	writef(dev, msg, args...)
}

// Info 输出信息日志
func Info(msg string, args ...any) {
	writef(info, msg, args...)
}

// Warn 输出警告日志
func Warn(msg string, args ...any) {
	writef(warn, msg, args...)
}

// Error 输出错误日志
func Error(msg string, args ...any) {
	writef(error, msg, args...)
}

// Fatal 输出致命错误日志并退出程序
func Fatal(msg string, args ...any) {
	writef(fatal, msg, args...)
	os.Exit(1)
}

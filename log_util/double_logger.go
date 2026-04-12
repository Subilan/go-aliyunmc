package log_util

import (
	"io"
	"log"
)

// doubleLogger 是同时向控制台和文件输出日志的 Logger 实现
type doubleLogger struct {
	console *log.Logger
	file    *log.Logger
}

// newDoubleLogger 创建一个新的 doubleLogger 实例，接受控制台输出和文件输出的 io.Writer，以及日志前缀和颜色
//  - consoleOut: 控制台输出的 io.Writer，通常是 os.Stdout 或 os.Stderr
//  - fileOut: 文件输出的 io.Writer，通常是一个实现了 io.Writer 接口的文件写入器
//  - plainPrefix: 日志前缀，不包含颜色编码
//  - color: 用于控制台输出的颜色编码，例如 colorBlue、colorGreen 等
func newDoubleLogger(consoleOut io.Writer, fileOut io.Writer, plainPrefix string, color string) *doubleLogger {
	return &doubleLogger{
		console: newLogger(consoleOut, colorizePrefix(plainPrefix, color)),
		file:    newLogger(fileOut, plainPrefix),
	}
}

// Printf 同时向控制台和文件输出格式化的日志消息
func (l *doubleLogger) Printf(format string, v ...any) {
	l.console.Printf(format, v...)
	l.file.Printf(format, v...)
}
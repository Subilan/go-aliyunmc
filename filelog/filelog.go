package filelog

import (
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogWriter 返回一个 os.Stdout 与 lumberjack.Logger 组合而成的 io.Writer 用于输出日志内容
func NewLogWriter(filename string) io.Writer {
	return io.MultiWriter(os.Stdout, &lumberjack.Logger{
		Filename: fmt.Sprintf("./logs/%s.log", filename),
	})
}

// NewLogger 返回一个以 NewLogWriter 返回值为 writer 的 *log.Logger
func NewLogger(filename string, prefixName string) *log.Logger {
	return log.New(NewLogWriter(filename), "["+prefixName+"] ", log.LstdFlags)
}

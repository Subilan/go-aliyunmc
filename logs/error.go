package logs

import (
	"log"
	"os"
)

var ErrorLogger = log.New(os.Stdout, "[ERROR] ", log.LstdFlags)

// Error 输出错误日志
func Error(msg string, args ...any) {
	ErrorLogger.Printf(msg, args...)
}
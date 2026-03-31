package logs

import (
	"log"
	"os"
)

var InfoLogger = log.New(os.Stdout, "[INFO] ", log.LstdFlags)

// Info 输出信息日志
func Info(msg string, args ...any) {
	InfoLogger.Printf(msg, args...)
}
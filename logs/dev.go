package logs

import (
	"log"
	"os"
)

var DevLogger = log.New(os.Stdout, "[DEV] ", log.LstdFlags)

// Dev 输出调试日志
func Dev(msg string, args ...any) {
	DevLogger.Printf(msg, args...)
}
package env

import "os"

var DEV bool

// MustInitialize 初始化环境变量
func MustInitialize() {
	DEV = os.Getenv("GO_ALIYUNMC_DEV") == "1"
}

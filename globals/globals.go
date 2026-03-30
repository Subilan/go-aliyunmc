package globals

import "os"

var DEV bool

func MustInitialize() {
	DEV = os.Getenv("GO_ALIYUNMC_DEV") == "1"
}

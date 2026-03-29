package globals

import "os"

var DEV bool

func MustLoadGlobals() {
	DEV = os.Getenv("GO_ALIYUNMC_DEV") == "1"
}

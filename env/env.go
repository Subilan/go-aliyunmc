package env

import "os"

type Env struct {
	IsProduction bool
}

var env Env

func Load() {
	_, env.IsProduction = os.LookupEnv("GO_ALIYUNMC_PRODUCTION")
}

func IsProduction() bool {
	return env.IsProduction
}

package casbin

import "go-aliyunmc/utils"

var C Config

// Config Casbin配置
type Config struct {
	ModelPath  string `toml:"model_path" validate:"required" comment:"Casbin模型文件路径"`
	PolicyPath string `toml:"policy_path" validate:"required" comment:"Casbin策略文件路径"`
}

// MustLoadConfig 加载配置
func MustLoadConfig() {
	utils.MustBindConfig(&C, "casbin")
}

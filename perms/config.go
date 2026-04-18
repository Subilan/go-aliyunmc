package perms

import "go-aliyunmc/utils"

var C Config

// Config 是权限系统的相关配置
type Config struct {
	ModelPath  string `toml:"model_path" validate:"required" comment:"Casbin模型文件路径"`
	PolicyPath string `toml:"policy_path" validate:"required" comment:"Casbin策略文件路径"`
}

func init() {
	utils.MustBindConfig(&C, "perms")
}

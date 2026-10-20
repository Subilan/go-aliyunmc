package config

// Perms 对应 [perms]：权限系统的相关配置。
type Perms struct {
	ModelPath  string `toml:"model_path" default:"assets/rbac/model.conf" comment:"Casbin模型文件路径"`
	PolicyPath string `toml:"policy_path" default:"assets/rbac/policy.csv" comment:"Casbin策略文件路径"`
}

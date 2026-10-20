package perms

import (
	"github.com/Subilan/go-aliyunmc/internal/config"
	"github.com/Subilan/go-aliyunmc/internal/log_util"

	"github.com/casbin/casbin/v2"
)

var enforcer *casbin.Enforcer

// MustInitialize 初始化 Casbin 权限管理器
func MustInitialize() {
	var err error
	enforcer, err = casbin.NewEnforcer(config.Get().Perms.ModelPath, config.Get().Perms.PolicyPath)
	if err != nil {
		log_util.Fatal("无法初始化 Casbin 权限管理器: %v", err)
	}
}

func Enforce(role Role, obj, act string) (bool, error) {
	return enforcer.Enforce(role.Normal(), obj, act)
}

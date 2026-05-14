package perms

import (
	"github.com/Subilan/go-aliyunmc/log_util"

	"github.com/casbin/casbin/v2"
)

var enforcer *casbin.Enforcer

// MustInitialize 初始化 Casbin 权限管理器
func MustInitialize() {
	var err error
	enforcer, err = casbin.NewEnforcer(C.ModelPath, C.PolicyPath)
	if err != nil {
		log_util.Fatal("无法初始化 Casbin 权限管理器: %v", err)
	}
}

func Enforce(role Role, obj, act string) (bool, error) {
	return enforcer.Enforce(role.Normal(), obj, act)
}
package perms

import (
	"go-aliyunmc/log_util"
	"github.com/casbin/casbin/v2"
)

var En *casbin.Enforcer

// MustInitialize 初始化 Casbin 权限管理器
func MustInitialize() {
	enforcer, err := casbin.NewEnforcer(C.ModelPath, C.PolicyPath)
	if err != nil {
		log_util.Fatal("无法初始化 Casbin 权限管理器: %v", err)
	}
	En = enforcer
}

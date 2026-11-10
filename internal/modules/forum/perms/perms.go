// Package perms 持有论坛模块独立的 Casbin 策略。
// 角色与 lab 共用 perm_util.Role，具体策略在实现论坛功能时补充。
package perms

import (
	_ "embed"

	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
)

//go:embed policy.csv
var policyCSV string

var enforcer *perm_util.Enforcer

// MustInitialize 用编译进二进制的论坛策略初始化 Casbin 权限管理器。
func MustInitialize() {
	var err error
	enforcer, err = perm_util.NewWithPolicy(policyCSV)
	if err != nil {
		log_util.Fatal("无法初始化 forum 权限管理器: %v", err)
	}
}

// Enforce 判断 role 是否可以对 obj 执行 act。
func Enforce(role perm_util.Role, obj, act string) (bool, error) {
	return enforcer.Enforce(role.Normal(), obj, act)
}

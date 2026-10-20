package perms

import (
	_ "embed"

	"github.com/Subilan/go-aliyunmc/internal/log_util"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist/string-adapter"
)

//go:embed model.conf
var modelConf string

//go:embed policy.csv
var policyCSV string

var enforcer *casbin.Enforcer

// MustInitialize 用编译进二进制的模型与策略初始化 Casbin 权限管理器
func MustInitialize() {
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		log_util.Fatal("无法解析 Casbin 模型: %v", err)
	}

	enforcer, err = casbin.NewEnforcer(m, stringadapter.NewAdapter(policyCSV))
	if err != nil {
		log_util.Fatal("无法初始化 Casbin 权限管理器: %v", err)
	}
}

func Enforce(role Role, obj, act string) (bool, error) {
	return enforcer.Enforce(role.Normal(), obj, act)
}

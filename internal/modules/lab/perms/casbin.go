package perms

import (
	_ "embed"

	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"

	"github.com/gin-gonic/gin"
)

//go:embed policy.csv
var policyCSV string

var enforcer *perm_util.Enforcer

// MustInitialize 用编译进二进制的 lab 策略初始化 Casbin 权限管理器。
func MustInitialize() {
	var err error
	enforcer, err = perm_util.NewWithPolicy(policyCSV)
	if err != nil {
		log_util.Fatal("无法初始化 lab 权限管理器: %v", err)
	}
}

// Enforce 判断 role 是否可以对 obj 执行 act。
func Enforce(role perm_util.Role, obj, act string) (bool, error) {
	return enforcer.Enforce(role.Normal(), obj, act)
}

// CanRequest 判断该权限等级是否可以以 c.Request.Method 方法访问 c.Request.URL.Path 路径
func CanRequest(role perm_util.Role, c *gin.Context) (bool, error) {
	return Enforce(role, c.Request.URL.Path, c.Request.Method)
}

// CanExecute 判断该权限等级是否可以对 target 进行 execute 操作
func CanExecute(role perm_util.Role, target string) (bool, error) {
	return Enforce(role, target, "execute")
}

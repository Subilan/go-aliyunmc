package casbin

import (
	"log"

	"github.com/casbin/casbin/v2"
)

var En *casbin.Enforcer

// MustInitialize 初始化Casbin权限管理
func MustInitialize() {
	enforcer, err := casbin.NewEnforcer("casbin/rbac_model.conf", "casbin/rbac_policy.csv")
	if err != nil {
		log.Fatalf("Failed to initialize casbin enforcer: %v", err)
	}
	En = enforcer
}

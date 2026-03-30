package casbin

import (
	"log"

	"github.com/casbin/casbin/v2"
)

var En *casbin.Enforcer

// MustInitialize 初始化Casbin权限管理
func MustInitialize() {
	enforcer, err := casbin.NewEnforcer(C.ModelPath, C.PolicyPath)
	if err != nil {
		log.Fatalf("Failed to initialize casbin enforcer: %v", err)
	}
	En = enforcer
}

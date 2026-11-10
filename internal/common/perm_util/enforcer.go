package perm_util

import (
	_ "embed"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist/string-adapter"
)

// model.conf 是 lab 与 forum 共用的 Casbin 模型；各模块只提供自己的策略文件。
//
//go:embed model.conf
var modelConf string

// Enforcer 是 Casbin enforcer 的轻量封装，每个模块各持有一个独立实例。
type Enforcer struct {
	enforcer *casbin.Enforcer
}

// NewWithPolicy 用共享模型和模块自己的策略初始化一个 enforcer。
func NewWithPolicy(policyCSV string) (*Enforcer, error) {
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, err
	}

	e, err := casbin.NewEnforcer(m, stringadapter.NewAdapter(policyCSV))
	if err != nil {
		return nil, err
	}

	return &Enforcer{enforcer: e}, nil
}

// Enforce 判断 subject 是否可以对 obj 执行 act。
func (e *Enforcer) Enforce(sub, obj, act string) (bool, error) {
	return e.enforcer.Enforce(sub, obj, act)
}

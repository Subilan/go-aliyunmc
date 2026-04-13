package tasks

import (
	"go-aliyunmc/casbin"
	"go-aliyunmc/h"
	"net/http"
)

type TaskEnforcerFunc func(string, map[string]any) error

// enforceRuleIfArgsMatch 是一个辅助函数，它完成 sub=role、obj=rule、act=execute 的权限检查，且仅在 condition(args) 返回 true 时才执行检查。
//  - role 是执行者角色，决定了权限检查的主体（sub）。
//  - condition 是一个函数，用于检查任务参数是否满足特定条件。仅当 condition 返回 true 时，才会进行权限检查。
//  - rule 是一个字符串，表示权限检查的对象（obj）。权限检查将基于执行者角色、该对象和固定的动作 act=execute 来进行。
func enforceRuleIfArgsMatch[T any](role string, rule string, args map[string]any, condition func(T) bool) error {
	var params T

	if err := ShouldBindArgs(args, &params); err != nil {
		return h.HttpError(http.StatusBadRequest, "无法解析参数："+err.Error())
	}

	if !condition(params) {
		return nil
	}

	sub := role
	obj := rule
	act := "execute"

	ok, err := casbin.En.Enforce(sub, obj, act)

	if err != nil {
		return err
	}

	if !ok {
		return h.HttpError(http.StatusForbidden, "权限不足")
	}

	return nil
}

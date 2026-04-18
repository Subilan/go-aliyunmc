package tasks

import "go-aliyunmc/store/models"

// getTaskDefinitionAndCheck 用于获取任务定义并执行对权限、参数的检查。user为nil时表示系统。
func getTaskDefinitionAndCheck(user *models.User, taskType models.TaskType, args map[string]any) (*TaskDefinition, error) {
	def := GetTaskDefinition(taskType)

	if def == nil {
		return nil, ErrTaskTypeNotFound
	}

	// 对任务类型的权限检查
	if user != nil && def.Role.Gt(user.Role) {
		return nil, ErrPermissionDenied
	}

	// 对具体任务参数的权限检查
	if user != nil && def.E != nil {
		if err := def.E(user.Role, args); err != nil {
			return nil, err
		}
	}

	// 对任务参数的检查
	if def.C != nil {
		if err := def.C(args); err != nil {
			return nil, err
		}
	}

	return def, nil
}

// TriggerTask 执行任务触发流程：查找定义、执行前置校验、启动执行器。user 可以为 nil 表示系统。
func TriggerTask(taskType models.TaskType, user *models.User, args map[string]any) (*Executor, *models.Task, error) {
	def, err := getTaskDefinitionAndCheck(user, taskType, args)

	if err != nil {
		return nil, nil, err
	}

	executor := NewExecutor(def)

	var task *models.Task
	if user != nil {
		task, err = executor.RunTask(&user.ID, args)
	} else {
		task, err = executor.RunTask(nil, args)
	}

	if err != nil {
		return nil, nil, err
	}

	return executor, task, nil
}

// TriggerTaskSync 与 TriggerTask 类似，但会阻塞到任务结束。user 可以为 nil 表示系统。
func TriggerTaskSync(taskType models.TaskType, user *models.User, args map[string]any) (*models.Task, error) {
	def, err := getTaskDefinitionAndCheck(user, taskType, args)

	if err != nil {
		return nil, err
	}

	executor := NewExecutor(def)

	var task *models.Task
	if user != nil {
		task, err = executor.RunTaskAndWait(&user.ID, args)
	} else {
		task, err = executor.RunTaskAndWait(nil, args)
	}

	return task, err
}

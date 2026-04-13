package tasks

import "go-aliyunmc/store/models"

// getTaskDefinitionAndCheck 是 TriggerTask 和 TriggerTaskSync 的公共部分：获取任务定义、执行检查函数和权限检查函数。user 可以为 nil 表示系统，此时跳过权限检查。
func getTaskDefinitionAndCheck(user *models.User, taskType models.TaskType, args map[string]any) (*TaskDefinition, error) {
	def := GetTaskDefinition(taskType)

	if def == nil {
		return nil, ErrTaskTypeNotFound
	}

	if def.C != nil {
		if err := def.C(args); err != nil {
			return nil, err
		}
	}

	if def.E != nil && user != nil {
		if err := def.E(user.Role, args); err != nil {
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

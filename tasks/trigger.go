package tasks

import "go-aliyunmc/store/models"

func getTaskDefinitionAndCheck(taskType models.TaskType, args map[string]any) (*TaskDefinition, error) {
	def := GetTaskDefinition(taskType)

	if def == nil {
		return nil, ErrTaskTypeNotFound
	}

	if def.C != nil {
		if err := def.C(args); err != nil {
			return nil, err
		}
	}

	return def, nil
}

// TriggerTask 执行任务触发流程：查找定义、执行前置校验、启动执行器。
func TriggerTask(taskType models.TaskType, by *uint, args map[string]any) (*Executor, *models.Task, error) {
	def, err := getTaskDefinitionAndCheck(taskType, args)

	if err != nil {
		return nil, nil, err
	}

	executor := NewExecutor(def)
	task, err := executor.RunTask(by, args)

	if err != nil {
		return nil, nil, err
	}

	return executor, task, nil
}

// TriggerTaskSync 与 TriggerTask 类似，但会阻塞到任务结束。
func TriggerTaskSync(taskType models.TaskType, by *uint, args map[string]any) (*models.Task, error) {
	def, err := getTaskDefinitionAndCheck(taskType, args)

	if err != nil {
		return nil, err
	}

	executor := NewExecutor(def)
	return executor.RunTaskAndWait(by, args)
}

package consts

// TaskType 表示任务的类型。任务的类型一定程度上介绍了该任务的作用所在。
type TaskType string

const (
	// TaskTypeInstanceDeployment 表示一个实例部署任务
	TaskTypeInstanceDeployment TaskType = "instance_deployment"
)

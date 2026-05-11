package tasks

import (
	"go-aliyunmc/perms"
	"go-aliyunmc/store/models"
	"time"
)

type TaskDefinition struct {
	// Timeout 定义了任务的超时时间。当任务执行时间超过该值时，任务将被自动标记为失败。0 表示无超时时间。
	Timeout time.Duration `json:"timeout"`

	// Type 定义了任务的类型，它是一个字符串，用于唯一标识一种任务。Type 与任务函数 F 一一对应，可以理解为任务函数的标识符。
	Type models.TaskType `json:"type"`

	// Exclusive 定义了任务是否属于独占类型。如果 Exclusive 为 true，则在该任务执行期间，同类型的其他任务将无法执行。
	Exclusive bool `json:"exclusive"`

	// Role 定义了执行该任务所需的权限等级，只有大于或等于该权限等级的用户可以执行此任务。如果不填，相当于 perms.RoleUser，表示所有登录用户都可以执行。
	Role perms.Role `json:"role"`

	// F 是任务的执行函数，它接受一个 TaskContext 作为参数，并返回一个 error。任务函数包含了任务的具体执行逻辑。
	//  - F 会在一个独立的 goroutine 中被调用，TaskContext 用于在任务执行过程中记录任务的状态、输出以及处理任务的中断等逻辑。
	//  - F 返回表示任务结束，其成功与否取决于返回值 error 是否为 nil。
	//  - 如果 F 返回一个非 nil 的 error 或者 panic，任务将被标记为失败，并且 error 的内容将被记录为任务的失败原因。
	//  - 如果 F 返回 nil，任务将被标记为成功。
	F TaskFunc `json:"-"`

	// C 是任务的检查函数，它接受一个 map[string]any 作为参数，并返回一个 error。
	// 检查函数用于在任务执行前验证输入参数以及环境的合法性。
	C TaskCheckFunc `json:"-"`

	// E 是任务的权限检查函数，它接受 role 和 map[string]any 作为参数，并返回一个 error。
	// 权限检查函数用于在任务执行前验证执行者是否具有足够的权限来执行带有相应参数的该任务。
	E TaskEnforcerFunc `json:"-"`
}

type TaskEnforcerFunc func(perms.Role, map[string]any) error

type TaskFunc func(*TaskContext, map[string]any) error

var TaskDefinitions = make(map[models.TaskType]*TaskDefinition)

func GetTaskDefinition(taskType models.TaskType) *TaskDefinition {
	def, ok := TaskDefinitions[taskType]
	if !ok {
		return nil
	}
	return def
}

func TryGetTaskDefinition(taskType string) (*TaskDefinition, bool) {
	def, ok := TaskDefinitions[models.TaskType(taskType)]
	return def, ok
}

func init() {
	TaskDefinitions[models.TaskTypeTest] = &TaskDefinition{
		Exclusive: true,
		Type:      models.TaskTypeTest,
		Timeout:   1 * time.Hour,
		F:         testTask,
	}

	TaskDefinitions[models.TaskTypeDeploy] = &TaskDefinition{
		Exclusive: true,
		Type:      models.TaskTypeDeploy,
		Timeout:   0, // 不设置超时，由任务内部逻辑控制
		F:         deployTask,
		C:         checkDeployTask,
	}

	TaskDefinitions[models.TaskTypeBackup] = &TaskDefinition{
		Exclusive: BackupC.Exclusive,
		Type:      models.TaskTypeBackup,
		Timeout:   time.Duration(BackupC.TimeoutSec) * time.Second,
		F:         backupTask,
		C:         checkBackupTask,
		Role:      perms.RoleOperator,
	}

	TaskDefinitions[models.TaskTypeArchive] = &TaskDefinition{
		Exclusive: true,
		Type:      models.TaskTypeArchive,
		Timeout:   time.Duration(ArchiveC.TimeoutSec) * time.Second,
		F:         archiveTask,
		C:         checkArchiveTask,
		Role:      perms.RoleOperator,
	}

	TaskDefinitions[models.TaskTypeCreateInstance] = &TaskDefinition{
		Exclusive: true,
		Type:      models.TaskTypeCreateInstance,
		Timeout:   10 * time.Minute,
		F:         createInstanceTask,
		C:         checkCreateInstanceTask,
		E:         enforceCreateInstanceTask,
	}

	TaskDefinitions[models.TaskTypeStartServer] = &TaskDefinition{
		Exclusive: true,
		Type:      models.TaskTypeStartServer,
		Timeout:   3 * time.Minute,
		F:         startServerTask,
		C:         checkStartServerTask,
	}
}

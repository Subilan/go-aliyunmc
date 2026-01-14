package consts

// InstanceStatus 表示一个实例可能处于的状态。大部分实例状态的枚举值与阿里云文档中的保持一致，参考：https://api.aliyun.com/api/Ecs/2014-05-26/DescribeInstanceStatus
type InstanceStatus string

const (
	// InstanceRunning 表示实例正在运行。正在运行的实例不一定可以执行操作，因为操作系统可能正在初始化。处于初始化阶段的实例也不可被删除。
	InstanceRunning InstanceStatus = "Running"
	// InstanceStopping 表示实例正在关闭中
	InstanceStopping InstanceStatus = "Stopping"
	// InstanceStopped 表示实例未运行
	InstanceStopped InstanceStatus = "Stopped"
	// InstancePending 表示实例正在被创建
	InstancePending InstanceStatus = "Pending"
	// InstanceStarting 表示实例正在开启中
	InstanceStarting InstanceStatus = "Starting"
	// InstanceUnableToGet 是用于监控器中的一个状态，表示实例的状态无法从阿里云的接口获取。
	InstanceUnableToGet InstanceStatus = "UnableToGet"
	// InstanceInvalid 是系统的一个私有状态，一般表示实例已经不存在。通常，一个不存在的实例不应该有状态，但在一些特殊的情况下，实例已经不存在但仍然记录了它的信息，无效状态的用途便在此。
	//
	// 例如，当实例被外部删除（如在阿里云控制台被删除，或者被阿里云释放）时，系统中可能仍然保存了实例的动态信息，此时实例的状态应当转换为“无效”。
	InstanceInvalid InstanceStatus = ""
)

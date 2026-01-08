package consts

type InstanceStatus string

const (
	InstanceRunning  InstanceStatus = "Running"
	InstanceStopping InstanceStatus = "Stopping"
	InstanceStopped  InstanceStatus = "Stopped"
	InstancePending  InstanceStatus = "Pending"
	InstanceStarting InstanceStatus = "Starting"
	InstanceInvalid  InstanceStatus = ""
)

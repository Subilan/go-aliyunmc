package consts

type CommandExecutionStatus string

const (
	CommandExecSuccess CommandExecutionStatus = "success"
	CommandExecError   CommandExecutionStatus = "error"
	CommandExecCreated CommandExecutionStatus = "created"
)

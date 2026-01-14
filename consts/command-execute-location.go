// Package consts 集中了系统中可能用到的一些常量和枚举。
package consts

// CommandExecuteLocation 表示一个 commands.Command 可能执行的位置。
type CommandExecuteLocation string

const (
	// ExecuteLocationServer 表示在 Minecraft 服务器内指令，i.e. 服务器指令，如 stop、op、deop
	ExecuteLocationServer CommandExecuteLocation = "server"

	// ExecuteLocationShell 表示在实例的 Shell 中执行，i.e. 系统指令，如 rm -rf /*
	ExecuteLocationShell CommandExecuteLocation = "shell"
)

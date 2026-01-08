package consts

type CommandType string

const (
	CmdTypeStartServer    CommandType = "start_server"
	CmdTypeStopServer     CommandType = "stop_server"
	CmdTypeBackupWorlds   CommandType = "backup_worlds"
	CmdTypeArchiveServer  CommandType = "archive_server"
	CmdTypeScreenfetch    CommandType = "screenfetch"
	CmdTypeGetServerSizes CommandType = "get_server_sizes"
)

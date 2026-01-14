package consts

// CommandType 是 commands.Command 的类型。指令的类型同时也可作为指令的名称使用，或者更确切地说，指令的类型也就是指令的标识符。
type CommandType string

const (
	// CmdTypeStartServer 是开启服务器的指令
	CmdTypeStartServer CommandType = "start_server"
	// CmdTypeStopServer 是关闭服务器的指令
	CmdTypeStopServer CommandType = "stop_server"
	// CmdTypeBackupWorlds 是备份服务器的世界存档的指令
	CmdTypeBackupWorlds CommandType = "backup_worlds"
	// CmdTypeArchiveServer 是归档服务器文件的指令
	CmdTypeArchiveServer CommandType = "archive_server"
	// CmdTypeScreenfetch 是获取实例运行 screenfetch 信息的指令
	CmdTypeScreenfetch CommandType = "screenfetch"
	// CmdTypeGetServerSizes 是获取服务器存档文件大小信息的指令（基于 du）
	CmdTypeGetServerSizes CommandType = "get_server_sizes"
	// CmdTypeGetServerProperties 是获取服务器 server.properties 文件的非敏感字段的指令（基于 cat）
	CmdTypeGetServerProperties CommandType = "get_server_properties"
	// CmdTypeGetOps 是获取服务器 ops.json 文件内容的指令（基于 cat）
	CmdTypeGetOps CommandType = "get_ops"
	// CmdTypeGetCachedPlayers 是获取服务器 cached_players.json 文件内容的指令（基于 cat）
	CmdTypeGetCachedPlayers CommandType = "get_cached_players"
)

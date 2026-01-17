package config

// ServerConfig 包括了与 Minecraft 服务器相关信息的配置项
type ServerConfig struct {
	// Port 是游戏的端口地址，用于查询在线玩家等信息。默认一般为 25565
	Port uint16 `toml:"port" validate:"required" comment:"MC服务器地址，默认25565"`

	// RconPort 是游戏的 RCON 服务地址，用于发送指令。默认一般为 25575
	RconPort uint16 `toml:"rcon_port" validate:"required" comment:"MC服务器的RCON服务地址，默认25575，请务必保证server.properties中RCON已开启"`

	// RconPassword 是游戏的 RCON 密码，用于发送指令。
	RconPassword string `toml:"rcon_password" validate:"required" comment:"MC服务器的RCON密码，可在server.properties中设置和查看"`
}

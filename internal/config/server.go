package config

// Server 对应 [server]：Minecraft 服务端自身的配置。
type Server struct {
	// Port 是游戏的端口地址，用于查询在线玩家等信息。
	Port uint16 `toml:"port" default:"25565" comment:"MC服务器地址"`

	// RconPort 是游戏的 RCON 服务地址，用于发送指令。
	RconPort uint16 `toml:"rcon_port" default:"25575" comment:"MC服务器的RCON服务地址，请务必保证server.properties中RCON已开启"`

	// RconPassword 是游戏的 RCON 密码，用于发送指令。
	RconPassword string `toml:"rcon_password,commented" validate:"required" comment:"MC服务器的RCON密码，可在server.properties中设置和查看"`
}

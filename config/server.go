package config

// ServerConfig 包括了与 Minecraft 服务器相关信息的配置项
type ServerConfig struct {
	// Port 是服务器的游戏地址，用于查询在线玩家等信息。默认一般为 25565
	Port uint16 `toml:"port" validate:"required"`

	// RconPort 是服务器的 RCON 服务地址，用于发送指令。默认一般为 25575
	RconPort uint16 `toml:"rcon_port" validate:"required"`
}

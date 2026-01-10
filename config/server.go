package config

type ServerConfig struct {
	Port     uint16 `toml:"port" validate:"required"`
	RconPort uint16 `toml:"rcon_port" validate:"required"`
}

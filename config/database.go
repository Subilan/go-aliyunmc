package config

type DatabaseConfig struct {
	Host     string `toml:"host" validate:"required"`
	Port     int    `toml:"port" validate:"required"`
	Username string `toml:"username" validate:"required"`
	Password string `toml:"password" validate:"required"`
	Database string `toml:"database" validate:"required"`
}

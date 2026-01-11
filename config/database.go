package config

import "fmt"

type DatabaseConfig struct {
	Host     string `toml:"host" validate:"required"`
	Port     int    `toml:"port" validate:"required"`
	Username string `toml:"username" validate:"required"`
	Password string `toml:"password" validate:"required"`
	Database string `toml:"database" validate:"required"`
}

func (d DatabaseConfig) Dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local", d.Username, d.Password, d.Host, d.Port, d.Database)
}

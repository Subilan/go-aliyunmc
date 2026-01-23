package config

import "fmt"

// DatabaseConfig 包含了数据库相关的配置。目前仅支持 MySQL 一种数据库类型，欢迎贡献扩展。
type DatabaseConfig struct {
	// Host 是数据库的地址
	Host string `toml:"host" validate:"required" comment:"数据库主机"`

	// Port 是数据库的端口
	Port int `toml:"port" validate:"required" comment:"数据库端口"`

	// Username 是数据库的用户名。注意，请不要使用 root 用户
	Username string `toml:"username" validate:"required" comment:"数据库用户名，注意不要使用root"`

	// Password 是数据库用户的密码
	Password string `toml:"password" validate:"required" comment:"数据库用户密码"`

	// Database 是用于存储本系统数据的模式名
	Database string `toml:"database" validate:"required" comment:"存储本系统数据的模式名"`
}

// Dsn 返回数据库的数据源（用于连接数据库的字符串）。
func (d DatabaseConfig) Dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local", d.Username, d.Password, d.Host, d.Port, d.Database)
}

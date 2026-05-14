package store

import (
	"fmt"
	"github.com/Subilan/go-aliyunmc/utils"
)

var C Config

func init() {
	utils.MustBindConfigToml(&C, "store")
}

// Config 数据库配置
type Config struct {
	// Driver 数据库驱动类型，支持 mysql, postgres, sqlite
	Driver string `toml:"driver" validate:"required,oneof=mysql postgres sqlite" comment:"数据库驱动类型"`
	// Host 数据库主机地址
	Host string `toml:"host" validate:"omitempty" comment:"数据库主机地址"`
	// Port 数据库端口
	Port int `toml:"port" validate:"omitempty,min=1,max=65535" comment:"数据库端口"`
	// User 数据库用户名
	User string `toml:"user" validate:"omitempty" comment:"数据库用户名"`
	// Password 数据库密码
	Password string `toml:"password" validate:"omitempty" comment:"数据库密码"`
	// DBName 数据库名称
	DBName string `toml:"dbname" validate:"required" comment:"数据库名称"`
	// Charset 数据库字符集
	Charset string `toml:"charset" validate:"omitempty" comment:"数据库字符集"`
	// SSLMode 是否启用SSL
	SSLMode string `toml:"sslmode" validate:"omitempty,oneof=disable require verify-ca verify-full" comment:"SSL模式"`
	// Path SQLite数据库文件路径
	Path string `toml:"path" validate:"omitempty" comment:"SQLite数据库文件路径"`
}

// DSN 返回数据库连接字符串
func (s *Config) DSN() string {
	switch s.Driver {
	case "mysql":
		charset := s.Charset
		if charset == "" {
			charset = "utf8mb4"
		}
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			s.User, s.Password, s.Host, s.Port, s.DBName, charset)
	case "postgres":
		sslmode := s.SSLMode
		if sslmode == "" {
			sslmode = "disable"
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			s.Host, s.Port, s.User, s.Password, s.DBName, sslmode)
	case "sqlite":
		path := s.Path
		if path == "" {
			path = s.DBName + ".db"
		}
		return path
	default:
		return ""
	}
}

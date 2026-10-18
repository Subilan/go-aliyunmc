package config

import "fmt"

// Store 对应 configs/store.toml：数据库配置。
// 系统仅支持 MySQL（生产为 MariaDB）。
type Store struct {
	// Host 数据库主机地址
	Host string `toml:"host" default:"localhost" comment:"数据库主机地址"`
	// Port 数据库端口
	Port int `toml:"port" default:"3306" validate:"min=1,max=65535" comment:"数据库端口，MySQL 常用 3306"`
	// User 数据库用户名
	User string `toml:"user" default:"root" comment:"数据库用户名"`
	// Password 数据库密码
	Password string `toml:"password" comment:"数据库密码"`
	// DBName 数据库名称
	DBName string `toml:"dbname" default:"aliyunmc" comment:"数据库名称"`
	// Charset 数据库字符集
	Charset string `toml:"charset" default:"utf8mb4" comment:"数据库字符集"`
}

// DSN 返回 MySQL 连接字符串。
func (s *Store) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		s.User, s.Password, s.Host, s.Port, s.DBName, s.Charset)
}

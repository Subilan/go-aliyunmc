package config

import "fmt"

// Store 对应 configs/store.toml：数据库配置。
//
// 它单独放在一个文件里，是因为 run / migrate / create_user 三个入口都需要它，
// 而另外两个入口不该被云凭证之类的配置牵着走。
type Store struct {
	// Driver 数据库驱动类型，支持 mysql, postgres, sqlite
	Driver string `toml:"driver" default:"mysql" validate:"oneof=mysql postgres sqlite" comment:"数据库驱动类型"`
	// Host 数据库主机地址
	Host string `toml:"host" default:"localhost" comment:"数据库主机地址"`
	// Port 数据库端口。取值随驱动而变（mysql 常为 3306，postgres 常为 5432），
	// 因此不给默认值，而是要求 mysql/postgres 下必须显式填写——零值会生成 ":0"。
	Port int `toml:"port,commented" validate:"required_unless=Driver sqlite,max=65535" comment:"数据库端口，mysql 常用 3306，postgres 常用 5432"`
	// User 数据库用户名
	User string `toml:"user" default:"root" comment:"数据库用户名"`
	// Password 数据库密码
	Password string `toml:"password" comment:"数据库密码"`
	// DBName 数据库名称
	DBName string `toml:"dbname" default:"aliyunmc" comment:"数据库名称"`
	// Charset 数据库字符集，仅 mysql 使用
	Charset string `toml:"charset" default:"utf8mb4" comment:"数据库字符集，仅 mysql 使用"`
	// SSLMode 是否启用SSL，仅 postgres 使用
	SSLMode string `toml:"sslmode" default:"disable" validate:"oneof=disable require verify-ca verify-full" comment:"SSL模式，仅 postgres 使用"`
	// Path SQLite数据库文件路径，留空则使用 DBName + ".db"
	Path string `toml:"path" comment:"SQLite数据库文件路径，留空则使用 数据库名称.db"`
}

// DSN 返回数据库连接字符串。
//
// sqlite 的 path 依赖 DBName，是跨字段派生，tag 表达不了，因此保留在这里；
// charset 与 sslmode 的默认值则放在字段的 default tag 上，不再在此回填。
func (s *Store) DSN() string {
	switch s.Driver {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			s.User, s.Password, s.Host, s.Port, s.DBName, s.Charset)
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			s.Host, s.Port, s.User, s.Password, s.DBName, s.SSLMode)
	case "sqlite":
		if s.Path != "" {
			return s.Path
		}
		return s.DBName + ".db"
	default:
		return ""
	}
}

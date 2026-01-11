package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/Subilan/go-aliyunmc/config"
)

var Pool *sql.DB

// InitPool 初始化 Pool 变量并且进行 Ping 验证，以供系统运行过程中使用
func InitPool() error {
	var err error

	dbCfg := config.Cfg.Database
	Pool, err = sql.Open("mysql", dbCfg.Dsn())

	if err != nil {
		return err
	}

	Pool.SetConnMaxLifetime(0)
	Pool.SetMaxIdleConns(20)
	Pool.SetMaxOpenConns(20)

	return Pool.Ping()
}

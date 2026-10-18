package store

import (
	"fmt"
	"log"

	"github.com/Subilan/go-aliyunmc/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 是全局GORM连接池
var DB *gorm.DB

func Silent() *gorm.DB {
	return DB.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
}

// MustInitialize 初始化GORM连接池
func MustInitialize() {
	var err error

	// 配置GORM
	gormConfig := &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Info),
		TranslateError: true,
	}

	// 连接数据库
	DB, err = gorm.Open(mysql.Open(config.Get().Store.DSN()), gormConfig)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	fmt.Println("数据库连接成功!")
}

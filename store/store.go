package store

import (
	"fmt"
	"go-aliyunmc/store/models"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 是全局GORM连接池
var DB *gorm.DB

// MustInitialize 初始化GORM连接池
func MustInitialize(config StoreConfig) {
	var err error
	var dialector gorm.Dialector

	// 根据驱动类型选择对应的dialector
	switch config.Driver {
	case "mysql":
		dialector = mysql.Open(config.DSN())
	case "postgres":
		dialector = postgres.Open(config.DSN())
	case "sqlite":
		dialector = sqlite.Open(config.DSN())
	default:
		log.Fatalf("不支持的数据库驱动: %s", config.Driver)
	}

	// 配置GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// 连接数据库
	DB, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	fmt.Println("数据库连接成功!")
}

// AutoMigrate 自动迁移数据库表
func AutoMigrate() {
	DB.AutoMigrate(&models.User{})
}
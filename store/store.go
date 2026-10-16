package store

import (
	"fmt"
	"log"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/store/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
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
	var dialector gorm.Dialector

	// 根据驱动类型选择对应的dialector
	switch config.Get().Store.Driver {
	case "mysql":
		dialector = mysql.Open(config.Get().Store.DSN())
	case "postgres":
		dialector = postgres.Open(config.Get().Store.DSN())
	case "sqlite":
		dialector = sqlite.Open(config.Get().Store.DSN())
	default:
		log.Fatalf("不支持的数据库驱动: %s", config.Get().Store.Driver)
	}

	// 配置GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		TranslateError: true,
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
	DB.AutoMigrate(
		&models.User{},
		&models.Task{},
		&models.Instance{},
		&models.PlayerListSample{},
		&models.BalanceSample{},
		&models.UserPreference{},
		&models.Changelog{},
	)
}

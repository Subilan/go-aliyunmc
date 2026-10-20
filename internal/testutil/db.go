package testutil

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/store"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testMySQLHostEnv     = "GO_ALIYUNMC_TEST_MYSQL_HOST"
	testMySQLPortEnv     = "GO_ALIYUNMC_TEST_MYSQL_PORT"
	testMySQLUserEnv     = "GO_ALIYUNMC_TEST_MYSQL_USER"
	testMySQLPasswordEnv = "GO_ALIYUNMC_TEST_MYSQL_PASSWORD"

	defaultTestMySQLHost = "127.0.0.1"
	defaultTestMySQLPort = "3306"
	defaultTestMySQLUser = "root"

	testDBPrefix = "aliyunmc_test_"
)

// OpenMySQLTestDB 在测试 MySQL 服务器上创建临时数据库，应用版本化迁移后安装到 store.DB。
//
// 返回的 cleanup 会关闭连接并删除该数据库。连接信息取自 GO_ALIYUNMC_TEST_MYSQL_HOST、
// _PORT、_USER、_PASSWORD，未设置时分别默认为 127.0.0.1、3306、root、空密码；
// 账号需要 CREATE DATABASE 与 DROP DATABASE 权限。
func OpenMySQLTestDB() (cleanup func(), err error) {
	cfg, err := testMySQLConfig()
	if err != nil {
		return nil, err
	}

	dbName, err := randomTestDBName()
	if err != nil {
		return nil, err
	}

	serverDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("打开测试 MySQL 连接失败: %w", err)
	}
	if err := serverDB.Ping(); err != nil {
		_ = serverDB.Close()
		return nil, fmt.Errorf("连接测试 MySQL (%s) 失败: %w", cfg.Addr, err)
	}

	if _, err := serverDB.Exec("CREATE DATABASE `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"); err != nil {
		_ = serverDB.Close()
		return nil, fmt.Errorf("创建测试数据库 %s 失败: %w", dbName, err)
	}

	var db *gorm.DB
	cleanup = func() {
		if db != nil {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
			store.DB = nil
		}
		if _, err := serverDB.Exec("DROP DATABASE IF EXISTS `" + dbName + "`"); err != nil {
			fmt.Fprintf(os.Stderr, "清理测试数据库 %s 失败: %v\n", dbName, err)
		}
		_ = serverDB.Close()
	}

	dbCfg := *cfg
	dbCfg.DBName = dbName
	dsn := dbCfg.FormatDSN()

	if err := store.MigrateDSN(dsn); err != nil {
		cleanup()
		return nil, err
	}

	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		cleanup()
		return nil, err
	}

	store.DB = db
	return cleanup, nil
}

func testMySQLConfig() (*mysqlDriver.Config, error) {
	port := envOr(testMySQLPortEnv, defaultTestMySQLPort)
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("%s 不是合法端口: %q", testMySQLPortEnv, port)
	}

	cfg := mysqlDriver.NewConfig()
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(envOr(testMySQLHostEnv, defaultTestMySQLHost), port)
	cfg.User = envOr(testMySQLUserEnv, defaultTestMySQLUser)
	cfg.Passwd = os.Getenv(testMySQLPasswordEnv)
	cfg.ParseTime = true
	cfg.Loc = time.Local
	cfg.MultiStatements = true

	return cfg, nil
}

// randomTestDBName 生成进程间不冲突的库名，使并行运行的测试包互不干扰。
func randomTestDBName() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成测试数据库名失败: %w", err)
	}
	return testDBPrefix + hex.EncodeToString(buf), nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

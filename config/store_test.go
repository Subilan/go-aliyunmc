package config

import (
	"testing"

	"github.com/creasty/defaults"
)

// 这些用例先过一遍默认值填充，与运行时的加载顺序一致。
func TestStoreDSNDefaultValues(t *testing.T) {
	mysqlConfig := Store{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "",
		DBName:   "aliyunmc",
	}
	if err := defaults.Set(&mysqlConfig); err != nil {
		t.Fatalf("填充默认值失败: %v", err)
	}

	expectedMySQLDSN := "root:@tcp(localhost:3306)/aliyunmc?charset=utf8mb4&parseTime=True&loc=Local"
	if got := mysqlConfig.DSN(); got != expectedMySQLDSN {
		t.Errorf("MySQL DSN 生成错误，期望: %s, 实际: %s", expectedMySQLDSN, got)
	}

	postgresConfig := Store{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "",
		DBName:   "aliyunmc",
	}
	if err := defaults.Set(&postgresConfig); err != nil {
		t.Fatalf("填充默认值失败: %v", err)
	}

	expectedPostgresDSN := "host=localhost port=5432 user=postgres password= dbname=aliyunmc sslmode=disable"
	if got := postgresConfig.DSN(); got != expectedPostgresDSN {
		t.Errorf("PostgreSQL DSN 生成错误，期望: %s, 实际: %s", expectedPostgresDSN, got)
	}

	// sqlite 的 path 派生自 DBName
	sqliteConfig := Store{
		Driver: "sqlite",
		DBName: "aliyunmc",
	}
	expectedSQLiteDSN := "aliyunmc.db"
	if got := sqliteConfig.DSN(); got != expectedSQLiteDSN {
		t.Errorf("SQLite DSN 生成错误，期望: %s, 实际: %s", expectedSQLiteDSN, got)
	}
}

func TestStoreDSNCustomValues(t *testing.T) {
	mysqlConfig := Store{
		Driver:   "mysql",
		Host:     "db.example.com",
		Port:     3307,
		User:     "user",
		Password: "pass",
		DBName:   "mydb",
		Charset:  "utf8",
	}
	if err := defaults.Set(&mysqlConfig); err != nil {
		t.Fatalf("填充默认值失败: %v", err)
	}

	expectedMySQLDSN := "user:pass@tcp(db.example.com:3307)/mydb?charset=utf8&parseTime=True&loc=Local"
	if got := mysqlConfig.DSN(); got != expectedMySQLDSN {
		t.Errorf("MySQL DSN 生成错误，期望: %s, 实际: %s", expectedMySQLDSN, got)
	}

	postgresConfig := Store{
		Driver:   "postgres",
		Host:     "db.example.com",
		Port:     5433,
		User:     "user",
		Password: "pass",
		DBName:   "mydb",
		SSLMode:  "require",
	}
	if err := defaults.Set(&postgresConfig); err != nil {
		t.Fatalf("填充默认值失败: %v", err)
	}

	expectedPostgresDSN := "host=db.example.com port=5433 user=user password=pass dbname=mydb sslmode=require"
	if got := postgresConfig.DSN(); got != expectedPostgresDSN {
		t.Errorf("PostgreSQL DSN 生成错误，期望: %s, 实际: %s", expectedPostgresDSN, got)
	}

	sqliteConfig := Store{
		Driver: "sqlite",
		DBName: "mydb",
		Path:   "./data/mydb.sqlite",
	}
	expectedSQLiteDSN := "./data/mydb.sqlite"
	if got := sqliteConfig.DSN(); got != expectedSQLiteDSN {
		t.Errorf("SQLite DSN 生成错误，期望: %s, 实际: %s", expectedSQLiteDSN, got)
	}
}

package store

import (
	"testing"
)

// TestStoreConfigDefaultValues 测试StoreConfig的默认值
func TestStoreConfigDefaultValues(t *testing.T) {
	// 测试MySQL默认值
	mysqlConfig := StoreConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "",
		DBName:   "aliyunmc",
		Charset:  "", // 应该使用默认值utf8mb4
	}

	mysqlDSN := mysqlConfig.DSN()
	expectedMySQLDSN := "root:@tcp(localhost:3306)/aliyunmc?charset=utf8mb4&parseTime=True&loc=Local"
	if mysqlDSN != expectedMySQLDSN {
		t.Errorf("MySQL DSN 生成错误，期望: %s, 实际: %s", expectedMySQLDSN, mysqlDSN)
	}

	// 测试PostgreSQL默认值
	postgresConfig := StoreConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "",
		DBName:   "aliyunmc",
		SSLMode:  "", // 应该使用默认值disable
	}

	postgresDSN := postgresConfig.DSN()
	expectedPostgresDSN := "host=localhost port=5432 user=postgres password= dbname=aliyunmc sslmode=disable"
	if postgresDSN != expectedPostgresDSN {
		t.Errorf("PostgreSQL DSN 生成错误，期望: %s, 实际: %s", expectedPostgresDSN, postgresDSN)
	}

	// 测试SQLite默认值
	sqliteConfig := StoreConfig{
		Driver: "sqlite",
		DBName: "aliyunmc",
		Path:   "", // 应该使用默认值aliyunmc.db
	}

	sqliteDSN := sqliteConfig.DSN()
	expectedSQLiteDSN := "aliyunmc.db"
	if sqliteDSN != expectedSQLiteDSN {
		t.Errorf("SQLite DSN 生成错误，期望: %s, 实际: %s", expectedSQLiteDSN, sqliteDSN)
	}
}

// TestStoreConfigCustomValues 测试StoreConfig的自定义值
func TestStoreConfigCustomValues(t *testing.T) {
	// 测试MySQL自定义值
	mysqlConfig := StoreConfig{
		Driver:   "mysql",
		Host:     "db.example.com",
		Port:     3307,
		User:     "user",
		Password: "pass",
		DBName:   "mydb",
		Charset:  "utf8",
	}

	mysqlDSN := mysqlConfig.DSN()
	expectedMySQLDSN := "user:pass@tcp(db.example.com:3307)/mydb?charset=utf8&parseTime=True&loc=Local"
	if mysqlDSN != expectedMySQLDSN {
		t.Errorf("MySQL DSN 生成错误，期望: %s, 实际: %s", expectedMySQLDSN, mysqlDSN)
	}

	// 测试PostgreSQL自定义值
	postgresConfig := StoreConfig{
		Driver:   "postgres",
		Host:     "db.example.com",
		Port:     5433,
		User:     "user",
		Password: "pass",
		DBName:   "mydb",
		SSLMode:  "require",
	}

	postgresDSN := postgresConfig.DSN()
	expectedPostgresDSN := "host=db.example.com port=5433 user=user password=pass dbname=mydb sslmode=require"
	if postgresDSN != expectedPostgresDSN {
		t.Errorf("PostgreSQL DSN 生成错误，期望: %s, 实际: %s", expectedPostgresDSN, postgresDSN)
	}

	// 测试SQLite自定义值
	sqliteConfig := StoreConfig{
		Driver: "sqlite",
		DBName: "mydb",
		Path:   "./data/mydb.sqlite",
	}

	sqliteDSN := sqliteConfig.DSN()
	expectedSQLiteDSN := "./data/mydb.sqlite"
	if sqliteDSN != expectedSQLiteDSN {
		t.Errorf("SQLite DSN 生成错误，期望: %s, 实际: %s", expectedSQLiteDSN, sqliteDSN)
	}
}

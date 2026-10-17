package store

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/migrations"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const (
	// migrationDriver 是目前唯一支持版本化迁移的数据库。
	migrationDriver = "mysql"
	// migrationsTable 是 golang-migrate 记录已应用版本的默认表名。
	migrationsTable = "schema_migrations"
)

// Migrate 按版本顺序应用 migrations/mysql/ 下嵌入的 SQL 迁移。
//
// 旧版本的数据库由 GORM AutoMigrate 创建，没有版本记录。为了平滑接入，
// 首次运行时如果检测到已有 users 表但还没有 schema_migrations，就把基线
// 版本直接写入版本表，只执行基线之后的迁移。
func Migrate() error {
	driverName := config.Get().Store.Driver
	if driverName != migrationDriver {
		return fmt.Errorf("版本化迁移目前只支持 mysql，当前驱动为 %s", driverName)
	}

	src, err := iofs.New(migrations.FS, migrationDriver)
	if err != nil {
		return fmt.Errorf("加载 mysql 迁移文件失败: %w", err)
	}

	db, err := openMigrationDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// 基线检测必须在 database driver 初始化之前完成：WithInstance 会创建
	// schema_migrations 表，之后就无法区分新库和旧库了。
	needsBaseline := !DB.Migrator().HasTable(migrationsTable) && DB.Migrator().HasTable("users")

	dbDriver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", src, migrationDriver, dbDriver)
	if err != nil {
		return err
	}

	if needsBaseline {
		version, err := baselineVersion()
		if err != nil {
			return err
		}
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("写入迁移基线失败: %w", err)
		}
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	// 数据迁移与 schema 迁移在同一次 migrate 中执行。当前 schema 版本决定哪些
	// 数据迁移已经可以运行；数据迁移本身必须幂等（见 migrations/data/）。
	version, _, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		version = 0
	} else if err != nil {
		return fmt.Errorf("读取当前迁移版本失败: %w", err)
	}

	if err := applyDataMigrations(db, uint64(version)); err != nil {
		return err
	}

	return nil
}

// baselineVersion 返回旧库首次接入版本化迁移时应记录的版本号：
//   - 若库中已有 task_output_lines（本分支曾用 AutoMigrate 跑过），
//     基线为创建该表的迁移；
//   - 否则基线为最初的 baseline 迁移。
func baselineVersion() (uint64, error) {
	suffix := "_baseline"
	if DB.Migrator().HasTable(&models.TaskOutputLine{}) {
		suffix = "_task_output_lines"
	}

	entries, err := fs.ReadDir(migrations.FS, migrationDriver)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".up.sql")
		if name == entry.Name() || !strings.HasSuffix(name, suffix) {
			continue
		}

		versionPart, _, ok := strings.Cut(name, "_")
		if !ok {
			continue
		}

		version, err := strconv.ParseUint(versionPart, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("无法解析迁移版本 %q: %w", entry.Name(), err)
		}
		return version, nil
	}

	return 0, fmt.Errorf("在 migrations/%s 中找不到 %s 迁移", migrationDriver, suffix)
}

// openMigrationDB 打开一个仅用于执行迁移的 MySQL 连接。
// 一个迁移文件可能包含多条语句，因此需要显式开启 multiStatements。
func openMigrationDB() (*sql.DB, error) {
	dsn := config.Get().Store.DSN()
	if !strings.Contains(dsn, "multiStatements=") {
		separator := "?"
		if strings.Contains(dsn, "?") {
			separator = "&"
		}
		dsn += separator + "multiStatements=true"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开迁移连接失败: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return db, nil
}

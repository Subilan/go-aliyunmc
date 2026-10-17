package store

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/Subilan/go-aliyunmc/log_util"
	"github.com/Subilan/go-aliyunmc/migrations"
)

const dataMigrationSuffix = ".addendum.sql"

// applyDataMigrations 执行 migrations/data 下的数据迁移。
//
// 文件名以对应的 schema 迁移版本号开头，例如：
//
//	migrations/mysql/20261017062544_task_output_lines.up.sql
//	migrations/data/20261017062544_task_output_lines.addendum.sql
//
// 只有版本号不大于当前 schema 版本的数据迁移才会执行，因此它们和 schema 迁移
// 在同一次 ./aliyunmc migrate 中自动完成。
//
// 数据迁移没有独立的版本记录表：每次 migrate 都会重新执行，SQL 必须自身幂等
// （NOT EXISTS、INSERT IGNORE、带条件的 UPDATE 等），并且只包含 DML。
func applyDataMigrations(db *sql.DB, schemaVersion uint64) error {
	entries, err := fs.ReadDir(migrations.FS, "data")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("读取数据迁移目录失败: %w", err)
	}

	type dataMigration struct {
		name    string
		version uint64
	}

	items := make([]dataMigration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), dataMigrationSuffix) {
			continue
		}

		version, err := dataMigrationVersion(entry.Name())
		if err != nil {
			return err
		}
		items = append(items, dataMigration{name: entry.Name(), version: version})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].version == items[j].version {
			return items[i].name < items[j].name
		}
		return items[i].version < items[j].version
	})

	for _, item := range items {
		if item.version > schemaVersion {
			continue
		}

		body, err := fs.ReadFile(migrations.FS, "data/"+item.name)
		if err != nil {
			return fmt.Errorf("读取数据迁移 %s 失败: %w", item.name, err)
		}

		rows, err := execDataMigration(db, string(body))
		if err != nil {
			return fmt.Errorf("执行数据迁移 %s 失败: %w", item.name, err)
		}
		if rows > 0 {
			log_util.Info("应用数据迁移 %s: %d 行", item.name, rows)
		}
	}

	return nil
}

func dataMigrationVersion(name string) (uint64, error) {
	versionPart, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("数据迁移文件名缺少版本号: %s", name)
	}

	version, err := strconv.ParseUint(versionPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无法解析数据迁移版本 %q: %w", name, err)
	}
	return version, nil
}

// execDataMigration 在事务中执行整个 SQL 文件。
// 连接已开启 multiStatements，因此一个文件可以包含多条 DML。
func execDataMigration(db *sql.DB, body string) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	result, err := tx.Exec(body)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

package migrations

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

// MariaDB 不支持 MySQL 8 专有的 utf8mb4_0900_ai_ci。Atlas 的 dev 库如果换回
// mysql/8，生成的迁移会带这个 collation，并在生产执行时直接失败。
func TestMigrationsUseMariaDBCompatibleCollation(t *testing.T) {
	entries, err := fs.ReadDir(FS, "mysql")
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		data, err := fs.ReadFile(FS, "mysql/"+entry.Name())
		if err != nil {
			t.Fatal(err)
		}

		if strings.Contains(string(data), "utf8mb4_0900_ai_ci") {
			t.Errorf("%s 使用了 MySQL 8 专有的 utf8mb4_0900_ai_ci；MariaDB 不支持", entry.Name())
		}
	}
}

// 数据迁移通过同名文件关联到 schema 迁移，例如
// 20261017062544_task_output_lines.addendum.sql ↔ 20261017062544_task_output_lines.up.sql。
func TestDataMigrationsMatchSchemaMigrations(t *testing.T) {
	schemaEntries, err := fs.ReadDir(FS, "mysql")
	if err != nil {
		t.Fatal(err)
	}

	schema := make(map[string]bool)
	for _, entry := range schemaEntries {
		if base, ok := strings.CutSuffix(entry.Name(), ".up.sql"); ok {
			schema[base] = true
		}
	}

	dataEntries, err := fs.ReadDir(FS, "data")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return
		}
		t.Fatal(err)
	}

	for _, entry := range dataEntries {
		if entry.IsDir() {
			continue
		}
		base, ok := strings.CutSuffix(entry.Name(), ".addendum.sql")
		if !ok {
			t.Errorf("数据迁移文件必须以 .addendum.sql 结尾: %s", entry.Name())
			continue
		}

		if !schema[base] {
			t.Errorf("数据迁移 %s 没有对应的 schema 迁移 %s.up.sql", entry.Name(), base)
		}
	}
}

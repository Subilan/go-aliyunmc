// Package migrations 嵌入版本化 SQL 迁移。
package migrations

import "embed"

// FS 包含 mysql 下的 schema 迁移和 data 下的一次性数据迁移。
//
//go:embed mysql/*.sql data/*.sql
var FS embed.FS

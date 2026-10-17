// Package migrations 嵌入版本化 SQL 迁移。
package migrations

import "embed"

// FS 包含 mysql 下的 schema 迁移和 data 下的一次性数据迁移。
//
// atlas.sum 用于 Atlas 校验 mysql 迁移目录，不参与运行时嵌入；
// golang-migrate 通过文件名解析版本，不需要该文件。
//
//go:embed mysql/*.sql data/*.sql
var FS embed.FS

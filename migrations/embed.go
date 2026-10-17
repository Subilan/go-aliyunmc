// Package migrations 嵌入版本化 SQL 迁移。
package migrations

import "embed"

// FS 包含 mysql 目录下的 *.sql 迁移文件。
//
// atlas.sum 用于 Atlas 校验迁移目录，不参与运行时嵌入；
// golang-migrate 通过文件名解析版本，不需要该文件。
//
//go:embed mysql/*.sql
var FS embed.FS

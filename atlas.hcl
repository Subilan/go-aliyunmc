# Atlas 配置：从 store/models 的 GORM 实体生成 MySQL 差量 SQL 迁移。
# 详见 migrations/README.md。
#
# 注意：tools/atlasloader 是一个独立的 Go module，避免把 Atlas provider
# 的依赖（各数据库驱动、云 SDK）引入主程序。

variable "legacy" {
  type        = bool
  default     = false
  description = "生成 pre-versioning 基线时排除 task_output_lines；日常 diff 保持 false"
}

data "external_schema" "gorm_mysql" {
  program = [
    "go", "-C", "tools/atlasloader", "run", "-mod=mod", ".",
    "--legacy=${var.legacy}",
  ]
}

env "mysql" {
  src = data.external_schema.gorm_mysql.url
  dev = "docker://mysql/8/dev"

  migration {
    dir    = "file://migrations/mysql"
    format = golang-migrate
  }
}

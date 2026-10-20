# Atlas 配置：从 internal/store/models 的 GORM 实体生成 MySQL 差量 SQL 迁移。
# 详见 internal/migrations/README.md。
#
# 注意：internal/migrations/atlasloader 是一个独立的 Go module，避免把 Atlas provider
# 的依赖（各数据库驱动、云 SDK）引入主程序。

variable "legacy" {
  type        = bool
  default     = false
  description = "生成 pre-versioning 基线时排除 task_output_lines；日常 diff 保持 false"
}

data "external_schema" "gorm_mysql" {
  program = [
    "go", "-C", "internal/migrations/atlasloader", "run", "-mod=mod", ".",
    "--legacy=${var.legacy}",
  ]
}

env "mysql" {
  src = data.external_schema.gorm_mysql.url
  # 生产与本地开发都用 MariaDB；dev 库必须与目标服务器一致，否则生成的
  # collation（例如 MySQL 8 的 utf8mb4_0900_ai_ci）在 MariaDB 上无法执行。
  dev = "docker+maria://_/mariadb:10.11/dev"

  migration {
    dir    = "file://internal/migrations/mysql"
    format = golang-migrate
  }
}

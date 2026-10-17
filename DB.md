# 数据库变更与迁移

本文说明项目的数据库访问框架、schema 的管理方式，以及修改实体后如何生成和应用迁移。

## 基本框架

| 组件 | 职责 |
|---|---|
| GORM（`store/models/`） | 定义实体与 schema 的唯一事实来源；运行时读写数据库 |
| Atlas CLI + `atlas-provider-gorm` | 读取 GORM 实体，与已有迁移做差量，自动生成 SQL |
| golang-migrate | 运行时按版本顺序执行 SQL，并在 `schema_migrations` 表记录已应用版本 |

当前版本化迁移只支持 **MySQL**。`store.Migrate()` 检测到其他驱动（如 sqlite/postgres）会直接返回错误。
测试使用临时 SQLite 数据库，由 `store.DB.AutoMigrate(models.Entities()...)` 建表，不经过迁移文件。

生产实际运行的是 **MariaDB**（Debian 12 自带 10.11），因此 Atlas 的 dev 数据库也必须是 MariaDB：
`atlas.hcl` 使用 `docker+maria://_/mariadb:10.11/dev`。如果用 MySQL 8 作为 dev 库，生成的 DDL 会带
`utf8mb4_0900_ai_ci`，而 MariaDB 不支持该 collation，迁移会在生产执行时直接失败。

相关文件：

```
atlas.hcl                     # Atlas 配置（MariaDB dev 库）
tools/atlasloader/            # 独立 Go module，把 GORM 模型输出为 MySQL DDL
migrations/
  embed.go                    # //go:embed mysql/*.sql
  README.md                   # 迁移目录的详细说明
  mysql/
    <version>_<name>.up.sql
    <version>_<name>.down.sql
    atlas.sum                 # Atlas 完整性校验
store/migrate.go              # store.Migrate()：运行时应用迁移 + 旧库基线
store/models/entities.go      # Entities()：参与迁移的模型列表
```

## 为什么不再使用 AutoMigrate

`AutoMigrate` 直接对比模型与数据库并修改 schema，问题在于：

- 变更没有历史，无法在部署前 review 或回滚；
- 多实例/多环境之间的实际 schema 容易漂移；
- 对已有大表做隐式变更风险不可控。

改为版本化迁移后，模型仍然决定“目标 schema”，但落到数据库的每一步都先变成可提交、可审计的 SQL。

## 修改数据库的流程

### 1. 修改 GORM 模型

在 `store/models/` 中修改或新增实体。

新增实体时，必须把它加入 `store/models/entities.go` 的 `Entities()`：

```go
func Entities() []any {
    return []any{
        &User{},
        &Task{},
        // ...
        &YourNewModel{},
    }
}
```

`tools/atlasloader` 通过这个列表读取 schema；遗漏会导致 Atlas 看不到新表。

### 2. 生成差量迁移

```bash
make migrate-diff NAME=add_your_change
```

等价于：

```bash
atlas migrate diff --env mysql add_your_change
```

Atlas 会：

1. 启动临时 MariaDB 10.11 dev 数据库（Docker）；
2. 在该库中回放 `migrations/mysql/` 的已有迁移；
3. 通过 `tools/atlasloader` 读取当前 GORM 模型；
4. 对比两者，把差量写入新的 `<version>_<name>.up.sql` / `.down.sql`。

生成前需要本机有 `atlas` CLI，并确保 Docker 正在运行。

### 3. Review 并提交

检查 `.up.sql` / `.down.sql`：

- 是否有意外的 `DROP`/`ALTER`；
- 索引、默认值、字符集是否符合预期；
- down 迁移是否能安全回滚。

确认无误后提交迁移文件。如果手工改过 SQL，需要重新计算校验和：

```bash
atlas migrate hash --env mysql
```

### 4. 应用迁移

```bash
./aliyunmc migrate
```

迁移 SQL 已通过 `//go:embed` 编译进二进制，部署时不需要额外文件。
`migrate` 只读取 `configs/store.toml`，不会加载云凭证、SSH 私钥等配置。

生产部署时不需要手工执行：production 仓库的 deploy workflow 会在切换 `current` 之前，
用暂存目录里的新二进制自动执行 `migrate`。迁移失败会中止 workflow，线上 release 和
`current` 均保持不变。因此迁移应当保持 additive，旧二进制要能在新 schema 上继续运行。

## 数据迁移（SQL）

一次性数据回填放在 `migrations/data/`，文件名与对应的 schema 迁移同名，只把后缀换成
`.addendum.sql`：

```
migrations/
  mysql/20261017062544_task_output_lines.up.sql
  data/20261017062544_task_output_lines.addendum.sql
```

`store.Migrate()` 在 `m.Up()` 之后读取当前 schema 版本，并执行所有版本号不大于它的数据迁移。
因此它们和 schema 迁移在同一次 `./aliyunmc migrate`（也就是 production deploy workflow 的
迁移步骤）中自动完成。

约定：

- 数据迁移**没有独立版本表**：每次 migrate 都会重新执行，SQL 必须幂等
  （`NOT EXISTS`、`INSERT IGNORE`、带条件的 `UPDATE` 等）。
- 只放 DML，不要放 DDL；DDL 会让事务失去原子性。
- 文件必须能对应到 `migrations/mysql/` 下的 `<name>.up.sql`，由 `migrations` 包的测试守护。
- 数据迁移不参与 Atlas 的差量和 `atlas.sum`；Atlas 只管理 `migrations/mysql/`。

当前的回填 `20261017062544_task_output_lines.addendum.sql` 把旧版 `tasks.output` 作为单行
（`seq=1`）写入 `task_output_lines`，保留原始换行。`GetTaskOutputString` 会在行尾补一个
`\n`，所以重建结果与旧版一致。只处理已经结束的任务；发布期间仍在运行的任务会在后续
`migrate` 时补上。

## 版本与基线

已应用的版本记录在 `schema_migrations` 表。正常升级时，golang-migrate 会按版本号从小到大执行尚未应用的迁移。

项目从 `AutoMigrate` 切换到版本化迁移时，已有数据库没有版本记录。`store.Migrate()` 的处理方式：

| 数据库状态 | 行为 |
|---|---|
| 全新数据库（没有 `users` 表） | 从第一个迁移开始执行 |
| 旧库（有 `users`，没有 `schema_migrations`，没有 `task_output_lines`） | 基线记为 `baseline`，只执行其后的迁移 |
| 已被旧 `AutoMigrate` 建过 `task_output_lines` | 基线记为 `task_output_lines`，不重复建表 |

基线检测只会在首次接入时生效；一旦 `schema_migrations` 存在，后续升级都走正常的版本比较。

### 当前迁移

| 版本 | 内容 |
|---|---|
| `..._baseline` | 引入版本化迁移前的完整 schema |
| `..._task_output_lines` | 任务输出按行分表 |

## 常见操作

```bash
# 查看迁移目录是否与模型一致（输出 "no changes" 表示一致）
atlas migrate diff --env mysql check

# 手工修改 SQL 后重新计算校验和
atlas migrate hash --env mysql

# 查看当前数据库版本
mysql -e "SELECT * FROM schema_migrations"
```

## 注意事项

- **不要**在生产路径调用 GORM `AutoMigrate`。
- **不要**修改已经应用到任何环境的迁移文件；新增一个版本即可，否则已部署环境无法按序升级。
- 迁移应保持 additive / expand-contract：发布健康检查失败时会回滚二进制，但不会回滚数据库 schema。
- 每个迁移文件可能包含多条 SQL，运行时连接会自动附加 `multiStatements=true`。
- 迁移失败时 golang-migrate 会把 `schema_migrations` 标记为 dirty。此时需要先手工确认并修复数据库状态，再按需执行 `migrate force <version>`（CLI），不要直接删除版本表。
- 运行迁移前建议先备份数据库。
- 更详细的迁移目录说明见 `migrations/README.md`。

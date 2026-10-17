# 数据库迁移（MySQL）

本目录保存版本化的 MySQL 迁移。数据库结构的唯一事实来源仍然是 `store/models` 中的 GORM 实体，
但变更不再通过 `AutoMigrate` 直接作用到数据库，而是：

1. 用 **Atlas**（`atlas migrate diff`）对比 GORM 模型与已有迁移，自动生成差量 SQL；
2. 迁移文件提交到本目录；
3. 运行时由 `./aliyunmc migrate` 使用 **golang-migrate** 按版本顺序应用。

> 版本化迁移目前只支持 MySQL。其他数据库驱动运行 `./aliyunmc migrate` 会直接报错。

## 目录结构

```
migrations/
  mysql/
    <version>_<name>.up.sql
    <version>_<name>.down.sql
    atlas.sum       # Atlas 校验和，不参与运行时嵌入
  embed.go          # //go:embed mysql/*.sql，供运行时使用
```

当前版本：

| 版本 | 内容 |
|---|---|
| `..._baseline` | 引入版本化迁移前的完整 schema |
| `..._task_output_lines` | 任务输出分表 |

## 生成一次变更

1. 修改 `store/models/` 中的实体。**新增模型时同时把它加入 `models.Entities()`**
   （`tools/atlasloader` 通过这个列表读取 schema）。
2. 生成 MySQL 差量迁移：

   ```bash
   make migrate-diff NAME=add_some_column
   ```

   等价于：

   ```bash
   atlas migrate diff --env mysql add_some_column
   ```

3. 检查生成的两个文件（`.up.sql` / `.down.sql`），确认 SQL 符合预期。

## 手工调整迁移文件

Atlas 生成的 SQL 可以直接提交。如果确实需要手工修改，修改后必须重新计算校验和：

```bash
atlas migrate hash --env mysql
```

## 运行时应用

```bash
./aliyunmc migrate
```

- 迁移文件通过 `//go:embed` 编译进二进制，部署时不需要额外文件。
- 已应用的版本记录在 `schema_migrations` 表。
- 首次升级到版本化迁移时，如果检测到库中已有 `users` 表但没有 `schema_migrations`，
  会把基线版本直接写入版本表，然后只执行基线之后的迁移：
  - 旧库（没有 `task_output_lines`）→ 基线为 `baseline`；
  - 已用旧版 `AutoMigrate` 建过 `task_output_lines` 的库 → 基线为 `task_output_lines`。

## 工具依赖

- [Atlas CLI](https://atlasgo.io)（`atlas`）
- Docker：Atlas 用它启动临时 MySQL dev 数据库来计算差量
- `tools/atlasloader` 是一个独立 Go module，避免把 Atlas provider 的依赖带进主程序

## 注意

- 不要再调用 GORM 的 `AutoMigrate` 修改 production schema（测试中的临时 SQLite 库除外）。
- 不要在已有迁移文件上叠加变更；应新增一个迁移版本，保证已部署环境可以按序升级。
- 运行 `./aliyunmc migrate` 前建议先备份数据库。

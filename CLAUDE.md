# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / Test / Run

```bash
make gen-config   # Generate example configs into examples/
make test         # Run all tests (go test -v ./...)
go test ./...     # 直接运行即可，不需要先生成或复制配置文件
make build        # Build binary as aliyunmc
make bundle OUT=/tmp/bundle   # Build the linux/amd64 release bundle consumed by production
make dev          # Run in dev mode (sets GO_ALIYUNMC_DEV=1; auto-migrates DB)
make run          # Run without dev mode
make migrate-diff NAME=add_xxx   # Generate a MySQL diff migration from GORM models (Atlas + Docker, MariaDB 10.11 dev)
./aliyunmc migrate   # Apply pending versioned migrations
./aliyunmc check_config   # 只校验配置，不连库、不建客户端
./aliyunmc gen_policy [-config policy/policy.toml] [-env production]   # Render RAM policy to stdout
```

Run a single package test:
```bash
go test -v ./internal/routes/user_routes/
go test -v ./internal/config/ -run TestStoreDSN
```

需要数据库的测试会在 MySQL/MariaDB 上为每个测试包创建临时库（`aliyunmc_test_<随机>`），
应用嵌入的版本化迁移后在结束时删除。连接信息取自
`GO_ALIYUNMC_TEST_MYSQL_{HOST,PORT,USER,PASSWORD}`，默认
`127.0.0.1:3306` / `root` / 空密码；账号需要 `CREATE DATABASE` 与 `DROP DATABASE` 权限，
通常只需授予库级权限：`GRANT ALL PRIVILEGES ON \`aliyunmc_test_%\`.* TO '<user>'@'<host>';`。

API integration tests (requires running server + [hurl](https://hurl.dev)):
```bash
cd tests/api/users && make test-login USER=<name>
cd tests/api/users && make test-create USER=<name>
cd tests/api/tasks && make test-create-test-task USER=<name>
```

## Development Environment

本地开发直接使用 `.dev/config/`，并与生产使用同样的能力。隔离完全由配置决定：
开发配置指向独立的数据库、实例资源、SSH 密钥与 OSS 前缀，代码层不做限制。
`env.DEV` 只开启 Gin debug、`dev/dev` 自动登录和 GORM auto-migrate；
`make dev` 会对 `.dev/config/store.toml` 指向的库执行 AutoMigrate，运行前确认它指向开发库。
`.dev/` 含敏感数据且被 gitignore；运行时数据落在 `.dev/state/`。

## RAM Policy Generation

`aliyunmc gen_policy` renders `policy/templates/policy.json.tmpl` with the
values from `policy/policy.toml` and writes the result to stdout. The template
contains no account ID, region, bucket or policy name; those come from the
config. `policy/policy.example.toml` and the template are versioned;
`policy/policy.toml` is gitignored. `-env <name>` prints one environment as
pure JSON; without it all environments are printed with `#` headers.

## Architecture

This is a Go web application (Gin framework) for managing Alibaba Cloud ECS instances and Minecraft servers. The entry point is `cmd/aliyunmc/main.go`, which wires all modules together. All business packages live under `internal/`.

**Core module boundaries:**

| Package | Purpose |
|---|---|
| `cmd/aliyunmc/` | 主程序入口与子命令（`run`、`migrate`、`create_user`、`check_config`、`gen_policy`）。 |
| `cmd/configgen/` | 从 `internal/config` 的 tag 生成 `examples/`。 |
| `internal/routes/` | Route registration and HTTP handlers. Each sub-package has a `Bind(*gin.Engine)` and handler files prefixed `handle_`. |
| `internal/mid/` | Auth (`mid.Auth`) and RBAC permission (`mid.Perm`) middleware. Auth sets `user`/`user_id` in the Gin context; Perm checks Casbin rules. |
| `internal/h/` | Handler wrappers that enforce the `{"data": ...}` / `{"error": ...}` API contract. |
| `internal/store/` | GORM database layer. `store.DB` is the global MySQL connection pool. Models in `internal/store/models/`. Tests use a temporary MySQL database via `internal/testutil`. |
| `internal/tasks/` | Task execution system: definitions, execution lifecycle, SSE integration. `TaskDefinitions` map registers task types. |
| `assets/scripts/` | Shell script templates (`*.tmpl.sh`) executed remotely via SSH during tasks. |
| `assets/mc/` | Minecraft 语言 JSON 与其 embed 包，编译进二进制，不随 release 分发。 |
| `internal/sse/` | Server-Sent Events client: event framing, keep-alive, and the blocking write loop for a connection. |
| `internal/pubsub/` | Generic fan-out hub (`Hub[T]`) with per-subscriber buffers; shared by task event streams and state snapshots. |
| `internal/states/` | State store + `HubbedStore[T]` built on `pubsub.Hub`, broadcasting state snapshots to SSE clients. |
| `internal/aliyun/` | Alibaba Cloud SDK clients (ECS, VPC, OSS, BSS). Initialized once at startup. |
| `internal/perms/` | Casbin RBAC. Three roles: `basic` (empty string), `operator`, `superuser` — with inheritance (superuser > operator > basic). |
| `internal/monitors/` | Background goroutines: server/instance status polling, file sync, auto-archive idle, scheduled backup, spot instance interruption detection, best ECS candidate, player count / balance sampling. |
| `internal/remote_util/` | SSH connection, script template rendering (`text/template`), and remote command execution. Used by tasks and monitors. |
| `internal/config/` | 全部配置结构体、加载（严格解码 + 默认值 + 校验）与运行期 registry。其余包一律从 `config.Get()` 读取。 |
| `internal/global_states/` | Atomic global flags (e.g. `IsArchiving`) shared across packages to coordinate monitors and tasks. |
| `internal/log_util/` | `log/slog` based logging with named loggers, tint-colorized console output, and lumberjack file rotation. |
| `internal/env/` | Exposes `env.DEV` bool (set by `GO_ALIYUNMC_DEV=1`). |
| `internal/context_util/` | Helpers to extract `user`, `user_id`, and `role` from Gin context (set by `mid.Auth`). |
| `internal/utils/` | Project-root resolution for tests. |
| `internal/migrations/` | Go embed 包与版本化 SQL；不是运维脚本。 |

仓库级目录：`assets/rbac/` 与 `assets/scripts/` 随 release 发布，`assets/mc/` 编译进二进制；`policy/` 只用于生成 RAM 策略；`release/` 是与 production 仓库之间的发布契约；`tests/api/` 是 hurl 集成测试；`docs/` 是设计文档；`examples/` 是生成的示例配置；`.dev/` 是本地运行时 home。

## Initialization Order

初始化是显式发生的，不再依赖 `init()`：

- `config.MustLoad()` 一次性读入 `config/store.toml` + `config/config.toml`（严格解码 → 填默认值 → 校验）并安装进 registry。此后全程序通过 `config.Get()` 读取。
- `main()` 先调用 `env.MustInitialize()` — DEV flag。
- `runServer` 按以下顺序装配：
  1. `config.MustLoad()` — 全部配置
  2. `store.MustInitialize()` — 用 `config.Get().Store` 连接 MySQL
  3. `store.AutoMigrate()` — DEV 下由 GORM AutoMigrate 建表（生产仍走 `migrate`）
  4. `session.InitStore(store.DB)` — session 存储
  5. `perms.MustInitialize()` — Casbin enforcer
  6. `aliyun.MustInitialize()` — 阿里云 clients
  7. `remote_util.MustInitialize()` — 加载 `config.Get().SSH.PrivateKeyPath` 指向的私钥
  8. `tasks.RegisterTaskDefinitions()` — 任务定义注册
  9. `mc.MustLoadData()` — Minecraft 语言数据；`mc.Advancements()` 等访问器在首次访问时也会自动加载
  10. `monitors.MustInitialize(ctx)` — 启动各 monitor/sampler，需在路由注册之后
- `migrate` 和 `create_user` 只调用 `config.MustLoadStore()`（仅读 `store.toml`），因此不会被云凭证、私钥路径等其余各节挡住。
- `migrate` 调用 `store.Migrate()`：用嵌入的 `internal/migrations/mysql/*.sql`（golang-migrate 格式）应用版本化迁移。系统只支持 MySQL（生产为 MariaDB）；生产的 schema 变更必须通过 Atlas 生成迁移文件。DEV 模式由 `store.AutoMigrate()` 在启动时用 GORM AutoMigrate 补表。生产部署由 production 仓库的 workflow 在切换 `current` 前自动执行 `migrate`。
- `check_config` 调用 `config.MustLoad()` 后立即退出：不连库、不建客户端、不读私钥，用于部署前确认配置与二进制版本匹配。

各功能包只提供显式的 `MustInitialize()`，且只做副作用（连库、建 client、载 Casbin、读私钥）：配置由 `config.Get()` 自取，不再有包级配置全局，也没有 `MustLoadConfig()`。结构体上的 `validate` tags 会在加载时统一校验，无效配置直接终止进程。

Dev mode (`env.DEV == true`): auto-migrates DB schema, auto-creates a dev user, and runs Gin in debug mode.

## Handler Pattern

All route handlers return `(any, error)`. Use the appropriate wrapper:

```go
authorized.GET("/path", h.G(HandleSomething))        // basic handler: func(c *gin.Context) (any, error)
authorized.POST("/path", h.B(HandleWithBody))         // JSON body binding: func(req T, c *gin.Context) (any, error)
authorized.GET("/path", h.Q(HandleWithQuery))         // query param binding: func(q T, c *gin.Context) (any, error)
authorized.GET("/path", h.V(valueFn))                 // always 200: func() T
authorized.GET("/path", h.VE(valueFn))                // value + error: func() (T, error)
authorized.GET("/path", h.VB(valueFn))                // value + ok, 404 if false: func() (T, bool)
```

The `h.G` wrapper (used by all others) maps errors to HTTP status codes:
- `h.HttpError(code, msg)` → that status code
- `gorm.ErrRecordNotFound` → 404
- `gorm.ErrDuplicatedKey` → 409
- Everything else → 500

## Route Registration & RBAC

Each routes sub-package has a `Bind(*gin.Engine)` that registers paths on a router group. Protected routes chain `mid.Auth()` (session-based) and `mid.Perm()` (Casbin). Permissions are defined in `assets/rbac/policy.csv` using `keyMatch2` path matching.

Casbin model (`assets/rbac/model.conf`) and policy (`assets/rbac/policy.csv`) live under `assets/rbac/`. Role hierarchy: `superuser` > `operator` > `basic`.

The `perms.Role.Gt()` / `perms.Role.Gte()` methods provide ordered comparison for task-level permission checks (separate from route-level Casbin checks).

## Task System

Task types are registered in `internal/tasks/task_definition.go` via the `TaskDefinitions` map. Each `TaskDefinition` has:
- `F`: the task function (runs in a goroutine, receives `*TaskContext` + `args map[string]any`)
- `C`: optional pre-flight parameter validation
- `E`: optional parameter-level enforcer for fine-grained permissions
- `Role`: minimum role required
- `Exclusive`: whether only one instance of this type can run at a time
- `Timeout`: duration timeout (0 = no timeout, managed internally)

Task execution lifecycle:
1. `TriggerTask` performs role check, optional enforcer check, parameter validation, then creates an `Executor`
2. `Executor.RunTask` creates a DB record, creates a per-task `pubsub.Hub`, runs `F()` in a goroutine, and runs a `monitor()` goroutine that watches for completion/timeout/interrupt/cancellation
3. The task function communicates progress via `TaskContext`: `println(msg)`, `nextStep()`, `status(status)`, `done()`, `throw(err)`
4. `done()` → task marked success; `throw(err)` → task marked failed; timeout → `__TIMEOUT__`; interrupt → `__INTERRUPTED__`
5. On shutdown, `main()` calls `RangeExecutors` to interrupt all running tasks and waits for them to finish

Deploy steps are defined by the `deploySteps` slice in `internal/tasks/deploy.task.go` (each step names a script template or a local Go function, plus a per-step timeout). Backup/archive tasks are single-template based. Scripts are rendered from `.tmpl.sh` templates and executed over `remote_util`; the deploy task runs all of its steps on a single reused `remote_util.Conn`, while the single-template tasks use the one-shot `remote_util.Root(ip).ExecScript`.

Deploy has no destructive rollback: every step must be **idempotent** so a failed deploy can be retried directly. In particular, archive restore downloads into a temporary directory and atomically swaps it into place, and the data disk records its UUID in `/var/lib/aliyunmc/data-disk-uuid` so a retry mounts instead of reformatting. On failure the task error includes the step number/name, and `/var/lib/aliyunmc/deploy-state` records `status=failed` along with the failed step for on-instance inspection. `internal/tasks/deploy_script_test.go` renders every script template and runs `bash -n` as a syntax guard.

`oneclick_start_game`（启动游戏，`internal/tasks/oneclick_start_game.task.go`）是首页唯一的入口，并且是**幂等**的：每次触发先读取当前实例状态，只补做 `createInstance` → `deployTask` → `startServerTask` 中尚未完成的阶段（无实例时创建，未部署时部署，服务器未在线时启动）。因此用户始终只需点击「启动游戏」，失败后再次点击即可从失败点继续，不需要在「部署 / 启动服务器」之间切换。对外接口为 `oneclick_start_game` + 原有参数。`create_instance`、`deploy`、`start_server`、`oneclick_start_game` 共享 `TaskExclusiveGroupDeployChain`（`TaskDefinition.ExclusiveGroup`），在整条部署链路上互斥。

## SSH Remote Execution

`remote_util.Target` is the single entry point for every remote operation. Build one with `Root(ip)` or `Mc(ip)`, then use `Exec`, `ExecScript`, `Upload`, `DownloadFile`, `DownloadDir`, or `Probe`. The Target is what resolves credentials, timeouts, and retries; callers never choose a transport. Two transports live behind it:
- commands and scripts run through in-process `golang.org/x/crypto/ssh` with a line-by-line output callback
- file transfer runs a local `rsync -avz` subprocess

`Target.Connect` hands back a reusable `*remote_util.Conn`; multi-step callers (the deploy task) open one connection up front and run every step on it, while the one-shot `Exec`/`ExecScript` are connect-execute-close conveniences. The connection layer retries dialing (2s/4s backoff, honoring `ctx`) but returns immediately on errors that retrying cannot fix, such as authentication or algorithm-negotiation failures. A `Conn` whose transport dies is dropped, so the next operation re-dials instead of failing every remaining step.

`RenderScriptTemplate(templatePath, vars)` renders `.tmpl.sh` files with Go `text/template` and is transport-independent.

SSH always uses port 22. Authentication is role-based:
- `mc` uses the private key at `config.Get().SSH.PrivateKeyPath` (`[ssh]` in `config/config.toml`). The deploy task derives the matching public key from it via `remote_util.AuthorizedKeyLine()` and writes that into `/home/mc/.ssh/authorized_keys`, so the key material exists in exactly one place and cannot drift out of sync. The key must be passphrase-less because the service runs under systemd with no agent. rsync passes the same path via `-i` plus `-o IdentitiesOnly=yes`, so it can never fall back to an ambient key.
- `root` uses the password from `config.Get().Aliyun.Ecs.RootPassword`. This is needed only during deployment, before any key is authorized; root never gets an authorized key.

`config.Get().Aliyun.Ecs.ProdPassword` is still used to create the `mc` account, but no longer for SSH authentication.

## Config System

全部配置结构体定义在 `config` 包里，运行期通过 `config.Get()` 读取。默认目录是 `config/`，可由 `GO_ALIYUNMC_CONFIG_DIR` 覆盖。只有两个文件：

| 文件 | 内容 | 读取者 |
|---|---|---|
| `config/store.toml` | MySQL 数据库配置 | `run`、`migrate`、`create_user`、`check_config` |
| `config/config.toml` | 其余全部：`[base]`、`[aliyun]`、`[server]`、`[perms]`、`[ssh]`、`[user_routes]`、`[tasks.*]`、`[monitors.*]`、`[samplers.*]` | `run`、`check_config` |

`store` 单独成文件，是因为另外两个子命令只需要它。`Config.Store` 标了 `toml:"-"`，所以 `config.toml` 里若出现 `[store]` 会被严格解码直接报错，不会形成第二条真相。

加载顺序不可颠倒：严格解码（`DisallowUnknownFields`，未知或拼错的键直接报错）→ `creasty/defaults` 填默认值（只填零值）→ `go-playground/validator` 校验。字段 tag 的用法：

- `default:"..."` — 有合理默认值、且"零值即未设置"的字段
- `toml:"...,commented"` — 必填且没有合理默认值的字段（凭证、密钥路径、资源 ID 等），在示例里会被注释掉
- 数组与 `bool` 开关一律不给 `default`：显式的空列表会被静默替换、显式的 `false` 会被翻转
- `commented` 只能用在叶子字段上（用在表上会把整节连同子键一起注释掉），由 `internal/config/schema_test.go` 守护
- 分节路径与类型名严格对应（`[monitors.file_sync]` ↔ `MonitorsFileSync`），由 `TestSectionPaths` 守护

`cmd/configgen/main.go` 从这些 tag 生成 `examples/`（`make gen-config`）：comment 变注释、default 变默认值、commented 变注释掉的键。它不手写任何字面量，所以新增字段会自动出现在示例里。

Runtime `config/config.toml` and `config/store.toml` are user-managed — never auto-generate them. `examples/` is generated by `make gen-config` and versioned.

## Database Migrations

数据库 schema 由 `internal/store/models` 的 GORM 实体定义。系统只支持 MySQL（生产为 MariaDB）；生产变更通过版本化 SQL 管理：

- `internal/migrations/mysql/` 存放 golang-migrate 格式的迁移文件（`<version>_<name>.up.sql` / `.down.sql`），由 `internal/migrations/embed.go` 嵌入二进制。
- 修改模型后运行 `make migrate-diff NAME=...`（`atlas.hcl` + `internal/migrations/atlasloader`）自动生成差量 SQL；Atlas 通过 Docker 启动临时 **MariaDB 10.11** dev 库。不要换回 MySQL 8：它生成的 `utf8mb4_0900_ai_ci` 在 MariaDB 上无法执行。
- 新增 GORM 模型时必须把它加入 `internal/store/models/entities.go` 的 `Entities()`，否则 Atlas 看不到它。
- 运行时 `./aliyunmc migrate` 调用 `store.Migrate()`，用 golang-migrate 应用嵌入的迁移并记录到 `schema_migrations`。生产 deploy workflow 会在切换 `current` 之前自动执行这一步。
- 一次性数据迁移放在 `internal/migrations/data/`，文件名与同名 schema 迁移对应（`.addendum.sql`）。`store.Migrate()` 在 `m.Up()` 之后按 schema 版本执行它们；SQL 必须幂等，且只包含 DML。它们不参与 Atlas 差量和 `atlas.sum`。
- 旧数据库（没有 `schema_migrations`）首次运行时会按已有表结构自动写入基线版本，不会重复建表。
- 测试通过 `internal/testutil.OpenMySQLTestDB()` 在 MySQL 上创建临时库，并调用 `store.MigrateDSN()` 应用同一套迁移。
- 详情见 `internal/migrations/README.md`。不要再用 `AutoMigrate` 修改 production schema。

## SSE Streaming

Task output is streamed via SSE. The flow:
1. Client hits `GET /task/:id/output`
2. Handler looks up the running `Executor`, creates an `sse.Client`, and subscribes to the executor's `pubsub.Hub`
3. As the task calls `tc.println()`, messages flow through `Executor.monitor()` → hub broadcast → SSE client → browser. Each line gets a per-task monotonically increasing `Seq`; the event `id` is that sequence number. Output is buffered in `TaskContext.outputChan`, broadcast immediately, and persisted as rows in `task_output_lines` in batches (500ms / 256 lines). The legacy `tasks.output` column is kept for schema compatibility but is no longer updated (`store.UpdateTask` selects only mutable fields); `store.GetTask` reconstructs `Task.Output` from the lines for API compatibility.
4. On reconnect the browser sends `Last-Event-ID`. The handler replays persisted lines with `seq > Last-Event-ID` before resuming live events. For a finished task it replays from the database and sends `task_done` (whose ID is `lastSeq+1`); if the client already consumed that ID the handler returns 204 so `EventSource` stops reconnecting.
5. On terminal state the executor broadcasts `task_done`, then closes the hub; closing the subscriber channel ends `Client.Listen` and the connection

State watch endpoints (`/state/watch/*`) use the same `pubsub.Hub[T]` primitive with a buffer of 1, where a full buffer drops the value instead of the subscriber.

## Synced Remote Data

The file-sync monitor (`internal/monitors/file_sync.go`) periodically downloads files from the active ECS instance into `state/cache/`. Handlers that need this data (e.g. whitelist binding) read these JSON files directly at request time. The cache directory and sync schedule are configured in `[monitors.file_sync]` of `config/config.toml`.

## Conventions

- 配置结构体一律放在 `internal/config` 包（`internal/config/tasks.go`、`internal/config/monitors.go` …），一个域一个文件；字段用 `toml` + `validate` + `comment` + `default` tags。功能包不得再定义自己的配置结构体或包级配置全局。
- `config/` holds runtime config; `examples/` holds generated defaults.
- Dev mode (`GO_ALIYUNMC_DEV=1`): enables auto-migration, auto-creates dev user, runs in debug mode.
- Tests that need project-root relative files explicitly call `testutil.ChdirProjectRoot()` in `TestMain`; production code no longer changes the working directory.
- Script templates use `{{.VarName}}` Go template syntax and live under `assets/scripts/`.
- DB unique constraints are preferred over application-level checks for enforcing uniqueness (e.g. `WhitelistUUID` unique index).
- Every newly-created or just-updated API route should be configured correctly in `assets/rbac/policy.csv` to prevent access control issues.

## 代码注释

注释的唯一职责是补充代码本身无法表达的信息。写之前先问：删掉这条注释，读者会丢失什么？如果答案是「什么都不会」，就不要写。

### 该写什么

- **对外契约**：函数做什么、保证什么，例如阻塞语义、幂等性、并发安全、执行顺序。
- **不直观的原因**：为什么这样实现、绕开了什么问题、依赖什么前提。这类注释应放在对应代码旁边，而不是函数头。
- **非显然的约束与副作用**：超时、资源释放时机、错误语义中无法从错误值直接读出的部分。
- **导出符号的 Go doc**：首句以符号名开头，一句话说清职责。

### 不该写什么

- **别处已有单一事实来源的内容**：权限在 `assets/rbac/policy.csv`，路由在 `task_routes.go`，配置默认值在 `config` 的 tag 与 `examples/`。在注释里复述这些事实，等两处不同步时注释就变成了谎言。
- **可从名称、签名、类型或紧邻代码直接读出的信息**。不要为「完整」而把代码翻译成自然语言。
- **同一件事说两遍**。例如「等待任务函数退出后才返回」中的「后才返回」只是重复「等待」。
- **与函数职责无关的背景、用法示例、历史沿革和铺垫性说明**。

### 示例

反面：

```go
// HandleInterruptTask 中止正在执行的任务，并等待任务函数退出后才返回。
// 权限由 /task/:id/interrupt 的 Casbin 策略限制为 operator 及以上。
func HandleInterruptTask(c *gin.Context) (any, error)
```

正面：

```go
// HandleInterruptTask 中止正在执行的任务，并等待任务函数退出
func HandleInterruptTask(c *gin.Context) (any, error)
```

理由：第二句把 `assets/rbac/policy.csv` 中已有的权限事实复制进了 handler，权限调整后必然失配；而「后才返回」与「等待」表达的是同一件事。注释只保留代码无法表达的契约：这个 handler 会阻塞到任务函数退出。

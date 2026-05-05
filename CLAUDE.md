# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / Test / Run

```bash
make gen-config   # Generate example configs into example_configs/
make test         # Run all tests (go test -v ./...)
make build        # Build binary as aliyunmc
make dev          # Run in dev mode (sets GO_ALIYUNMC_DEV=1)
make run          # Run without dev mode
```

Run a single package test:
```bash
go test -v ./routes/user_routes/
go test -v ./store/ -run TestStoreConfig
```

API integration tests (requires running server + [hurl](https://hurl.dev)):
```bash
cd hurl/users && make test-login USER=<name>
cd hurl/users && make test-create USER=<name>
cd hurl/tasks && make test-create-test-task USER=<name>
```

## Architecture

This is a Go web application (Gin framework) for managing Alibaba Cloud ECS instances and Minecraft servers. The entry point is `main.go`, which wires all modules together.

**Core module boundaries:**

| Package | Purpose |
|---|---|
| `routes/` | Route registration and HTTP handlers. Each sub-package has a `Bind(*gin.Engine)` and handler files prefixed `handle_`. |
| `mid/` | Auth (`mid.Auth`) and RBAC permission (`mid.Perm`) middleware. Auth sets `user`/`user_id` in the Gin context; Perm checks Casbin rules. |
| `h/` | Handler wrappers that enforce the `{"data": ...}` / `{"error": ...}` API contract. |
| `store/` | GORM database layer. `store.DB` is the global connection pool (SQLite/MySQL/Postgres). Models in `store/models/`. |
| `tasks/` | Task execution system: definitions, execution lifecycle, SSE integration. `TaskDefinitions` map registers task types. |
| `scripts/` | Shell script templates (`*.tmpl.sh`) executed remotely via SSH during tasks. |
| `sse/` | Server-Sent Events broker and client for streaming task output to browsers. |
| `states/` | Generic pub/sub state hub (`Hub[T]`) for broadcasting state snapshots to SSE clients. |
| `aliyun/` | Alibaba Cloud SDK clients (ECS, VPC, OSS, BSS). Initialized once at startup. |
| `perms/` | Casbin RBAC. Three roles: `basic` (empty string), `operator`, `superuser` — with inheritance (superuser > operator > basic). |
| `monitors/` | Background goroutines: server/instance status polling, file sync, auto-archive idle, scheduled backup, best ECS candidate, player count / balance sampling. |
| `remote_util/` | SSH connection, script template rendering (`text/template`), and remote command execution. Used by tasks and monitors. |
| `global_states/` | Atomic global flags (e.g. `IsArchiving`) shared across packages to coordinate monitors and tasks. |
| `log_util/` | Structured logging with named loggers, file rotation, and ANSI-stripped file output. |
| `env/` | Exposes `env.DEV` bool (set by `GO_ALIYUNMC_DEV=1`). |
| `context_util/` | Helpers to extract `user`, `user_id`, and `role` from Gin context (set by `mid.Auth`). |
| `utils/` | Config loading (`MustBindConfigToml`) and project-root resolution for tests. |

## Initialization Order

`main.go`'s `init()` runs modules in this order, which is load-bearing:
1. `utils.MustBindConfigToml(&C, "main")` — main config (`config.go` at repo root, struct `Config`)
2. `store.MustInitialize()` — DB connection
3. `perms.MustInitialize()` — Casbin enforcer
4. `aliyun.MustInitialize()` — Alibaba Cloud clients
5. `env.MustInitialize()` — DEV flag

After `init()`, each package's own `init()` loads its TOML config from `configs/<name>.toml` via `utils.MustBindConfigToml`. Config structs use `validate` tags (go-playground/validator) that are enforced at startup — invalid config causes a fatal exit.

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

Each routes sub-package has a `Bind(*gin.Engine)` that registers paths on a router group. Protected routes chain `mid.Auth()` (session-based) and `mid.Perm()` (Casbin). Permissions are defined in `rbac_policy.csv` at repo root using `keyMatch2` path matching.

Casbin model (`rbac_model.conf`) and policy (`rbac_policy.csv`) are at repo root. Role hierarchy: `superuser` > `operator` > `basic`.

The `perms.Role.Gt()` / `perms.Role.Gte()` methods provide ordered comparison for task-level permission checks (separate from route-level Casbin checks).

## Task System

Task types are registered in `tasks/task_definition.go` via the `TaskDefinitions` map. Each `TaskDefinition` has:
- `F`: the task function (runs in a goroutine, receives `*TaskContext` + `args map[string]any`)
- `C`: optional pre-flight parameter validation
- `E`: optional parameter-level enforcer for fine-grained permissions
- `Role`: minimum role required
- `Exclusive`: whether only one instance of this type can run at a time
- `Timeout`: duration timeout (0 = no timeout, managed internally)

Task execution lifecycle:
1. `TriggerTask` performs role check, optional enforcer check, parameter validation, then creates an `Executor`
2. `Executor.RunTask` creates a DB record, starts an SSE broker, runs `F()` in a goroutine, and runs a `monitor()` goroutine that watches for completion/timeout/interrupt/cancellation
3. The task function communicates progress via `TaskContext`: `println(msg)`, `nextStep()`, `status(status)`, `done()`, `throw(err)`
4. `done()` → task marked success; `throw(err)` → task marked failed; timeout → `__TIMEOUT__`; interrupt → `__INTERRUPTED__`
5. On shutdown, `main()` calls `RangeExecutors` to interrupt all running tasks and waits for them to finish

Deploy tasks use a step-driven model via `[[steps]]` arrays in `task-deploy.toml` (each step: `name`, `script_path`, `timeout_sec`). Backup/archive tasks are single-template based (`template_path`). Scripts are rendered from `.tmpl.sh` templates and executed via `remote_util.ExecuteScriptRemote`.

## SSH Remote Execution

`remote_util/` provides the SSH layer used by tasks and monitors:
- `TryDialRoot(host, timeout)` — verify SSH connectivity
- `RenderScriptTemplate(templatePath, vars)` — render a `.tmpl.sh` file with Go `text/template`
- `ExecuteScriptRemote(scriptPath, ip, ctx, onLine, root)` — execute a local script on remote via SSH
- `ExecuteRemoteCommand(cmd, ip, ctx, onLine, root)` — execute a shell command remotely

SSH always uses port 22. User `root` with password from `aliyun.C.Ecs.RootPassword`, or user `mc` with `aliyun.C.Ecs.ProdPassword`.

## Config System

All config files live in `configs/*.toml` and are loaded via `utils.MustBindConfigToml[T any](ptr, name)`, which reads, unmarshals, and validates. `cmd/configgen/main.go` generates example configs into `example_configs/`. When changing config structs/tags, keep `configgen` and `example_configs/` aligned.

Runtime `configs/` files are user-managed — never auto-generate them.

The main config struct is `Config` in `config.go` (package main): port, CORS, AutoTLS, session keys. Other packages define their own config structs in `config.go` files within the package directory.

## SSE Streaming

Task output is streamed via SSE. The flow:
1. Client hits `GET /task/:id/output`
2. Handler looks up the running `Executor`, creates an `sse.Client`, subscribes to the executor's broker
3. As the task calls `tc.println()`, messages flow through `Executor.monitor()` → broker broadcast → SSE client → browser
4. On `task_done` event, both broker and client connection close

State watch endpoints (`/state/watch/*`) use a similar pattern but with `states.Hub[T]` for generic pub/sub instead of task brokers.

## Synced Remote Data

The file-sync monitor (`monitors/file_sync.go`) periodically downloads files from the active ECS instance into `remote_data_cache/`. Handlers that need this data (e.g. whitelist binding) read these JSON files directly at request time. The cache directory and sync schedule are configured in `configs/monitor-file-sync.toml`.

## Conventions

- Task config types live in `tasks/config.go`; template vars structs use `toml` + `validate` tags.
- `configs/` holds runtime config; `example_configs/` holds generated defaults.
- Dev mode (`GO_ALIYUNMC_DEV=1`): enables auto-migration, auto-creates dev user, runs in debug mode.
- Tests that rely on `store.DB` must run from the project root (handled by `utils.init()` which detects `go test` and chdirs).
- Script templates use `{{.VarName}}` Go template syntax and live under `scripts/`.
- DB unique constraints are preferred over application-level checks for enforcing uniqueness (e.g. `WhitelistUUID` unique index).

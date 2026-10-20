---
title: 构建与版本注入
description: Makefile 的常用目标与通过 ldflags 注入版本、提交和构建时间的方式。
---

# 构建与版本注入

## 常用目标

| 目标 | 作用 |
|---|---|
| `make build` | 编译当前平台二进制并写入版本信息 |
| `make bundle` | 生成 linux/amd64 发布包，`OUT` 指定输出目录 |
| `make dev` | 以 DEV 模式运行，状态目录为 `.dev` |
| `make run` | 以 `.dev` 状态目录运行，不设置 DEV 模式 |
| `make test` | 运行全部测试 |
| `make gen-config` | 生成 `examples/config.toml` 与 `examples/store.toml` |
| `make gen-policy` | 渲染 RAM 策略 |
| `make migrate-diff` | 生成数据库迁移，迁移名由 `MIGRATION_NAME` 指定 |

## 版本信息

`internal/version` 的三个变量由构建脚本通过 `-ldflags` 注入。

```
-X github.com/Subilan/go-aliyunmc/internal/version.Version=1.0.0
-X github.com/Subilan/go-aliyunmc/internal/version.Commit=<git sha>
-X github.com/Subilan/go-aliyunmc/internal/version.BuildTime=<RFC3339>
```

`make build` 从 `VERSION` 文件、`git rev-parse` 与当前 UTC 时间取值。`make dev` 等未注入的入口保持 `dev`、`unknown` 默认值。

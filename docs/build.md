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
| `make bump` | 升级版本并打标签，`BUMP` 指定 major/minor/patch 或版本号 |
| `make bump-push` | 同 `make bump`，并推送提交与标签 |
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

## 发布版本

`VERSION` 用 `x.y.z` 形式的 semver 记录当前版本，对应的 git 标签是同名的 `v<x.y.z>`。

```bash
make bump BUMP=minor          # 1.0.0 -> 1.1.0，提交并打 v1.1.0
make bump-push BUMP=1.2.3      # 指定版本，并推送提交与标签
bash release/version.sh patch --dry-run  # 只打印将要执行的步骤
```

脚本先确认 `VERSION` 没有未提交改动、目标标签不存在，然后写入新版本、提交 `chore(release): v<x.y.z>`、打注解标签。`make bump` 不动远端。

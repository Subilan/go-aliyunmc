---
title: 模块划分与 API 前缀
description: lab 与 forum 同进程运行的目录边界、共享与独立的部分，以及新增模块的方式。
---

# 模块划分与 API 前缀

lab（控制台）与 forum（论坛）在同一个 Go 进程中运行，代码按模块组织：

```
internal/
  common/          # 共用：配置、DB、共享 User、session 工厂、Casbin 模型、日志、HTTP 包装
  modules/
    lab/           # lab 业务：路由、任务、监控、阿里云、lab RBAC 策略
    forum/         # 论坛骨架：空路由组、独立 session 与 RBAC 策略
    entities.go    # 合并各模块的 GORM 实体，供 DEV AutoMigrate 与 Atlas 使用
```

依赖方向固定为 `modules/* → common`；`common` 不得 import 任何业务模块。

## 共享与独立

| 项目 | 共享方式 |
|---|---|
| 用户与凭据 | 同一张 `users` 表；注册只在 lab 提供 |
| 登录态 | 独立。lab 使用 `lab_session` cookie（Path `/lab-v1`），forum 使用 `forum_session`（Path `/forum-v1`） |
| RBAC | 共用 `users.role` 与 `internal/common/perm_util` 的 `Role`；各模块维护自己的策略文件与 enforcer，只有在确实需要时才新增角色枚举 |
| 数据库 | 同一个库、同一份迁移版本序列（`internal/common/migrations`） |
| 配置 | 同一个 `config.toml` / `store.toml`，由 `internal/common/config` 加载 |

## API 前缀

- lab：`/lab-v1/...`，由 `internal/modules/lab.Bind` 注册。
- forum：`/forum-v1/...`，由 `internal/modules/forum.Bind` 注册。
- 根路径 `GET /` 保留为进程级健康检查。

不提供旧路径兼容；前端与后端需要同批上线。

## 新增一个模块

1. 在 `internal/modules/<name>` 下创建入口：`Prefix`、`MustInitialize()`、`Bind(*gin.Engine)`。
2. 需要独立登录态时，用 `internal/common/session_util.New(cookieName, cookiePath)` 创建自己的 manager，
   并在 `cmd/aliyunmc/main.go` 的 `session_util.InitStore` 中注册。
3. 需要独立 RBAC 时，在模块内维护自己的 `policy.csv` 与 enforcer；Casbin 模型和角色类型
   共享自 `internal/common/perm_util`，只有在确实需要时才新增角色枚举。
4. 有自己的表时，创建 `modules/<name>/models` 并实现 `Entities()`，
   再由 `internal/modules/entities.go` 合并。
5. 在 `cmd/aliyunmc/main.go` 中调用模块的 `MustInitialize()` 与 `Bind()`。

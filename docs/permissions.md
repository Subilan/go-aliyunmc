# Permissions

## 角色继承关系

| 规则 | 含义 |
| --- | --- |
| operator -> basic | operator 拥有 basic 的全部权限 |
| superuser -> operator | superuser 拥有 operator 与 basic 的全部权限 |

## 路由权限（Casbin + mid.Perm）

说明：以下规则由 `mid.Perm()` 调用 `role.CanRequest()` 执行，使用 `rbac_model.conf` 的 `keyMatch2` 对路径进行匹配。

| 资源 | 方法 | 最低角色 | 含义 |
| --- | --- | --- | --- |
| /user/profile | GET | basic | 查看当前用户资料 |
| /user/logout | GET | basic | 当前用户登出 |
| /user | DELETE | basic | 当前用户注销自身账号 |
| /task/:id | GET | basic | 查看任务详情 |
| /task/definition/:taskType | GET | basic | 查看任务定义 |
| /task/:id/output | GET | basic | 订阅任务输出流 |
| /task/trigger | POST | basic | 触发任务（具体任务仍受任务层权限约束） |
| /state/snapshot/server-status | GET | basic | 获取服务器状态快照 |
| /state/watch/server-status | GET | basic | 订阅服务器状态变化 |
| /state/snapshot/instance-status | GET | basic | 获取实例状态快照 |
| /state/watch/instance-status | GET | basic | 订阅实例状态变化 |
| /monitor/auto-archive-idle/remaining-secs | GET | basic | 获取空服回收倒计时 |
| /instance/candidates | GET | basic | 获取实例候选列表 |
| /instance/best-candidate | GET | basic | 获取当前最佳实例候选 |
| /server/stop | GET | operator | 停止运行中的服务端 |
| /server/data | GET | operator | 读取服务端同步数据 |
| /instance/active | DELETE | operator | 删除当前活跃实例 |

## 任务权限（任务层）

说明：以下权限不依赖路由中间件，而是在任务触发链路中执行（`tasks/getTaskDefinitionAndCheck` 与任务自定义 enforcer）。

| 任务类型/能力 | 最低角色 | 位置 | 含义 |
| --- | --- | --- | --- |
| backup | operator | `tasks/task_definition.go` | 仅 operator 及以上可触发备份任务 |
| archive | operator | `tasks/task_definition.go` | 仅 operator 及以上可触发归档任务 |
| create-custom-instance (execute) | operator | `tasks/create_instance.task.go` + `rbac_policy.csv` | 创建实例任务在使用自定义参数时，需要 execute 权限 |

## 公开路由（不走权限中间件）

| 资源 | 方法 | 含义 |
| --- | --- | --- |
| /user/register | POST | 用户注册 |
| /user/login | POST | 用户登录 |

## 当前生效模型

- 默认拒绝（deny by default）：请求未命中任何 allow 规则时拒绝。
- 角色继承：由 `g, child, parent` 规则提供。
- 路径匹配：使用 `keyMatch2`，支持 `:param` 风格路径。

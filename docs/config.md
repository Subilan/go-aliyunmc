---
title: 配置结构与参数
description: config 包的分节映射、加载校验顺序，以及阿里云和实例筛选参数的取值影响。
---

# 配置结构与参数

## 文件与加载

配置文件位于 `$GO_ALIYUNMC_STATE_DIR/config`，未设置该变量时为本目录下的 `config`。

| 文件 | 内容 | 加载入口 |
|---|---|---|
| `store.toml` | 数据库连接 | `config.MustLoadStore()`，`migrate` 与 `create_user` 使用 |
| `config.toml` | 其余全部配置 | `config.MustLoad()`，`run` 使用 |

`load()` 的顺序固定为严格解码、填默认值、校验。解码禁止未知键，配置里拼错或残留已删除的字段会直接报错。`defaults.Set` 只填零值，配置里显式写过的值不会被覆盖。字段约束写在 `validate` tag 上。

`config.Get()` 返回进程内唯一且只读的配置，未加载时 panic。需要默认值兜底的库函数使用 `config.TryGet()`，测试与特殊入口使用 `config.Set()`。

## 分节映射

TOML 分节与 Go 类型的对应关系：

- 类型名是分节路径的驼峰形式，`[monitors.file_sync]` 对应 `MonitorsFileSync`
- 聚合字段名是顶层分节名，结构体字段名是该节里的键
- `config.Get().Monitors.FileSync.PollIntervalSec` 对应 `[monitors.file_sync]` 里的 `poll_interval_sec`
- 每个类型上方注明它对应的 TOML 路径

`config` 包负责配置的结构定义与加载。连库、创建客户端、加载 Casbin、读取私钥等副作用由各功能包自己负责，功能包只通过 `config.Get()` 读取配置。

## 阿里云参数

`[aliyun]` 与 `[aliyun.ecs]` 的字段大多直接对应 `CreateInstance` 的参数，取值参考 https://api.aliyun.com/api/Ecs/2014-05-26/CreateInstance。

### 地域

`region_id` 决定实例与 OSS 的内网连通性。实例与 bucket 不在同一地域时无法内网传输，而归档依赖内网传输。内网速度通常超过 100MB/s 且不计流量，一个常规大小的存档在 1 至 2 分钟内可以传完。系统默认且仅支持内网传输，所以地域一开始就要确定。更换地域需要新建 bucket 并走公网迁移，会产生流量费用，速度取决于两边带宽。不同地域的抢占式实例库存也有差异。

### 中止动作

`spot_interruption_behavior` 取 `Stop` 或 `Terminate`。

- `Stop` 让实例进入节省停机模式，CPU 与内存释放，云盘保留并继续计费。库存短缺时数据不会丢失，推荐使用。
- `Terminate` 直接释放实例并删除全部数据。使用它必须配置合理的备份与归档间隔，否则会大幅回档。

### 安全组

`security_group_id` 对应的安全组需要放行服务所在 IP 到 25565 与 25575 端口，否则后端无法 Ping 服务器、查询状态或发送 RCON 指令。其他端口的放行需求同样在安全组内设置。

安全组说明见 https://help.aliyun.com/zh/ecs/user-guide/overview-44。

### 其他字段

- `prod_password` 是生产用户 `mc` 的密码，该用户在部署时创建。根用户只在部署阶段使用。
- `system_disk.category` 与 `data_disk.category` 通常使用 `cloud_essd`。可选值包括 `cloud_efficiency`、`cloud_ssd`、`cloud_essd`、`cloud`、`cloud_auto`、`cloud_essd_entry`，取值不当可能导致创建失败。

## 最佳实例筛选

`[monitors.best_ecs_candidate.filters]` 控制候选实例的过滤。

| 字段 | 作用 |
|---|---|
| `max_trade_price` | 可接受的最高交易价格，单位 CNY，超过该值的实例被筛除 |
| `instance_type_exclusion` | 对实例规格名生效的正则，匹配的规格被筛除 |

规格排除主要用于去掉性能不佳或对 Minecraft 无意义的规格，例如共享型 `ecs.e*` 与大数据型 `ecs.d*`。常见配置：

```toml
instance_type_exclusion = "^ecs\\.(e|s6|xn4|n4|mn4|e4|t|d).*$"
```

`mem_choices` 与 `cpu_core_count_choices` 不提供默认值，必须显式配置，并且应使用常见的内存大小和与内存比匹配的 vCPU 数量。

## 示例配置

`examples/config.toml` 与 `examples/store.toml` 由结构体 tag 生成：

```bash
make gen-config
```

`comment` tag 变成注释，`default` tag 变成默认值，`commented` 选项把必须自行填写的字段注释掉，例如凭证、密钥路径与资源 ID。

数据库配置与迁移见 [数据库变更与迁移](DB.md)。

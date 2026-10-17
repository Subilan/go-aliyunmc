# 可被现成轮子替代的核心实现（只读分析，按收益递减）

## 1. [done] log_util（约 320 行）→ log/slog + lumberjack + tint
手写 doubleLogger / appendFileWriter / rotatingFileWriter / prefixed_logger。
问题：每次 Write 都 open/close，只有按大小轮转、无保留期与压缩，颜色硬编码，全局 writer。
方案：stdlib log/slog + gopkg.in/natefinch/lumberjack.v2 + github.com/lmittmann/tint，
保留 log_util 门面，调用方零改动。收益最高。

## 2. [done] leaderboard + playerdata 每请求全量重建 → 缓存
BuildLeaderboard / BuildRawLeaderboard 每请求 ReadDir 整个 player_stats 并逐文件 Unmarshal，
BuildPlayerNameMap 重复读白名单，BatchQueryOnlineDates 全表扫后 Go 侧 split。
最小方案：x/sync/singleflight + 短 TTL 缓存（ristretto 或 go-cache）。
彻底方案：file-sync 钩子落表，用 SQL 排序分页。成本低收益高。

## 3. 远端 OSS 操作 → OSS Go SDK + RAM STS
archive/backup/install-ossutil/restore 脚本依赖实例上装 ossutil，并把 AccessKeySecret
渲染进脚本、写进实例 /root/.ossutilconfig。主控已依赖 alibabacloud-oss-go-sdk-v2 与 credentials-go。
方案：控制面用 OSS SDK 直传直取，或给实例发 RAM STS 临时凭据。安全性收益大，成本中等。

## 4. [done] sse（255 行）→ 自研 + 抽出 pubsub.Hub
手写 broker 的 actor 循环与原 states.Hub 合并为泛型 pubsub.Hub：任务流用 buffer=100 并在满时移除慢订阅者，
状态流用 buffer=1 并丢弃中间值；Hub.Close 取代 task_done 特判。
任务输出改为有缓冲 channel（1024 行）+ 阻塞背压 + 按行批量落库（500ms / 256 行，写入 task_output_lines）。
任务输出事件带序号 ID，已实现 Last-Event-ID 重连回放：先订阅再请求 monitor 刷盘，
从数据库回放 seq > Last-Event-ID 的行后续接实时事件；已结束任务直接回放并在 task_done 后返回 204。
未整体替换为 tmaxmax/go-sse、r3labs/sse：它们的模型是长生命周期 server + topic，
与“每任务一条短流、结束时关闭”不匹配，适配成本高于收益。

## 5. tasks 执行器注册表与互斥 → x/sync
maps.go 两个 sync.Map 存运行中／独占执行器，Executor.Wait 手搓结果通道，RunTask 用 recover 兜 panic。
x/sync 的 errgroup / semaphore / singleflight 更直白。
不建议整体换 asynq、river：进程内可中断可订阅是有意设计。成本低。

## 6. rsync 外部进程 + 手写 SSH（已处理）
remote_util/rsync.go 调 rsync -e "ssh -o BatchMode=yes"，与其余密码认证路径不一致。
决定不替换为 pkg/sftp：控制器与 ECS 跨地域走公网，rsync 的 -z 与增量传输有实际价值。
改为统一 API 层：remote_util.Target（Root/Mc + Exec/ExecScript/Upload/Download*/Probe）
统一解析凭据、超时与重试，rsync 降为内部传输细节；rsync 的 ctx 中断错误现可被 errors.Is 识别。

## 7. 手写退避重试 → cenkalti/backoff 或 SDK RetryOptions
DialWithRetry 固定 2s/4s 且不响应 ctx；create_instance.task.go 的 startInstanceWithRetry 手搓 5s ticker。
aliyun dara runtime 自带 RetryOptions（现在处处传空 &dara.RuntimeOptions{}），交给 SDK 即可。
成本很低。

## 8. mapstructure → go-viper/mapstructure/v2 或 stdlib json
tasks/helper.go 的 ShouldBindArgs 用已归档的 mitchellh/mapstructure v1.5.0。
args 来自 JSON body，json.Marshal + DisallowUnknownFields 即可等价 ErrorUnused: true。低收益零成本。

## 9. configgen（311 行手写 map）→ 反射生成
手写 map 默认值与 config struct 双份维护，易漂移。
struct 已有 toml 与 comment tag，go-toml/v2 的 Marshal 原生支持 comment tag 输出。
可降到几十行，但不属核心运行路径。

## 10. 其余低收益项
- h.G 对未知错误把 err.Error() 原样回给客户端，建议 c.Error + 统一脱敏 handler。
- states.Hub / HubbedStore 与 global_states 手写 1s ticker 倒计时（可换 time.Timer），换库收益有限。
- playerdata 用逗号拼 player_names 做 LIKE（还分 mysql/sqlite 分支）本质是 schema 问题，应建关联表。
- mc/init.go 从工作目录读 JSON，可用 //go:embed。
- store/dao_* 分页与排序白名单重复三处，抽泛型 scope，不建议用停更的 GORM 分页插件。

## 不建议动
scs、casbin、validator、GORM、go-toml、gin-contrib/cors、mcutil、bcrypt 已是合适轮子。
h 包装器与 tasks 任务编排抽象虽自研但贴合业务，替换成本高于收益。

## 结论
真正值得动手：第 1 项（日志）、第 2 项（排行榜缓存）、第 3 项（OSS 凭据）、第 7 项（SDK 重试）。

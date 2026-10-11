# 优化二：排行榜与玩家数据的每请求全量重建改成 TTL 缓存

本文记录一次针对「排行榜接口每次请求都全量重建」的优化，包含改动内容、基准方法、实测数据与已知局限。数据均为真机实测，可复现（见文末）。

## TL;DR

- 改动前：`GET /server/leaderboard` 与 `/server/leaderboard/stat` 每次请求都重新读目录、读白名单、逐个玩家读并解析 stats / advancements JSON、再查两次数据库，然后排序返回。
- 改动后：计算结果按指标缓存 30 秒（TTL + 单飞去重），底层的文件读取与解析另有一层 30 秒缓存。
- 实测（120 名玩家语料，本机 10 核 Apple Silicon）：

| 场景 | 改动前 | 改动后 | 提升 |
| --- | --- | --- | --- |
| `achievements` 单次请求（稳态） | 51.44 ms | 8.13 µs | 约 6300 倍 |
| `minecraft_playtime` 单次请求（稳态） | 14.52 ms | 9.79 µs | 约 1480 倍 |
| 页面级（9 个榜单端点各 20 次，共 180 次） | 3257 ms | 2.04 ms | 约 1600 倍 |
| 20 个并发客户端请求同一榜单 | 284 ms | 70 ms | 约 4.1 倍（CPU 工作量 20 份降到 1 份） |

稳态延迟已经与玩家数量解耦：缓存命中时只有一次切片拷贝加排序，120 人和 12 人没有数量级差别。

## 一、改动前的成本结构

问题不在算法，而在「没有缓存、且每次请求都把远端数据当冷数据重新读一遍」。以 `achievements` 指标、N 名玩家为例，单次请求的固定开销是：

1. `os.ReadDir` 列出 `remote_data_cache/player_stats`；
2. 读 `whitelist.json` 并反序列化，构建 UUID 到名字的映射；
3. N 次 `player_advancements/<uuid>.json` 文件读取 + JSON 反序列化，统计完成度；
4. `user_preferences` 与 `users` 两次 SQL 查询，算出排行榜退出者集合；
5. 排序并返回。

N = 120 时，一次请求约 122 次文件系统操作与 120 次大 JSON 解析。`avg_move_distance` 更差：它先读一次 stats 算游戏时长，再读一次 stats 算移动距离，单请求 242 次读取。这些成本每来一个请求就要再付一次，而远端数据的更新周期其实是白名单 60 秒、stats 与 advancements 300 秒。

## 二、改动内容

引入 `internal/cache`（55 行），把两个成熟库组合成一个最小接口：

```go
// internal/cache
type Store[V any] struct {
	items *ttlcache.Cache[string, V] // github.com/jellydator/ttlcache/v3
	group singleflight.Group         // golang.org/x/sync/singleflight
}

func (s *Store[V]) GetOrLoad(key string, load func() (V, error)) (V, error)
```

- `ttlcache` 负责 TTL 与容量上限（LRU 淘汰），并关闭了「命中即续期」，让 TTL 是绝对过期而不是滑动过期；
- `singleflight` 保证同一 key 的并发请求只执行一次 `load`，其余调用等待并共享结果；
- `load` 返回错误时不写缓存，避免把「文件暂时不存在」之类的瞬时失败固化 30 秒。

缓存分两层。结果层在 `leaderboard`：`BuildLeaderboard` / `BuildRawLeaderboard` 拆成「计算」与「排序」两半，计算结果按 `metric:<指标>` / `raw:<statKey>` 缓存，容量 64；`order`（asc/desc）只影响排序，因此不参与缓存键，同一指标的升序与降序共享一次计算。数据层在 `playerdata`：UUID 列表、UUID 到名字的映射、玩家 stats 原始 JSON、成就完成度各自缓存，容量 1 / 1 / 1024 / 1024。

TTL 统一取 30 秒，刻意短于远端同步周期（60 / 300 秒），因此缓存不会带来额外的可见延迟。`ReadWhitelist` 保持不缓存，因为绑定白名单需要读到最新文件。

顺带修掉两个小问题：排序改为在缓存值的副本上做（`slice` 拷贝 + `slices.SortStableFunc`），不再有「排序会污染缓存」的风险；并列数值增加 UUID 次序作为次级键，名次不再随请求抖动。

## 三、基准方法

- 对照组：改动前的代码（`ce3f0c5`，即上一项日志优化完成时的版本）通过 `git archive` 导出到 `/tmp/mc-before`；实验组是当前工作树导出的 `/tmp/mc-after`。两份代码用完全相同的基准测试文件，直接调用包级函数（不经 gin、不计网络与 JSON 响应序列化），测的正是缓存要消除的那部分成本。
- 语料：12 人语料是仓库 `remote_data_cache/` 里现有的真实数据；120 人语料在此基础上复制真实文件、生成 108 个合成 UUID 与对应白名单条目，保证每个玩家的 stats / advancements 文件大小与真实数据一致。同时向 `player_list_samples` 写入 30 天 × N 名玩家的采样行，让 `login_days` / `join_streak` 的全表扫描贴近真实。
- 数据库用 SQLite（生产是 MySQL），每次测量前重建。
- 每个指标先测 1 次「首次调用」，再连续测 20 次并取 p50；环境为 macOS / Apple Silicon / 10 核 / Go 1.26.3。
- 两组各跑两轮验证稳定性，下面表格引用第一轮，第二轮结果一致（页面级总耗时 3.229 s vs 3.257 s，2.04 ms vs 1.63 ms）。

## 四、稳态单次请求延迟

### 12 名真实玩家

| 指标 | 改动前 p50 | 改动后 p50 | 倍数 |
| --- | --- | --- | --- |
| minecraft_playtime | 1.849 ms | 458 ns | 4037 |
| achievements | 5.156 ms | 375 ns | 13748 |
| distance | 1.475 ms | 416 ns | 3545 |
| mob_kills | 1.479 ms | 334 ns | 4427 |
| blocks_mined | 1.597 ms | 375 ns | 4259 |
| avg_move_distance | 2.888 ms | 458 ns | 6305 |
| login_days | 471 µs | 542 ns | 868 |
| join_streak | 471 µs | 500 ns | 942 |
| raw: minecraft:walk_one_cm | 1.438 ms | 333 ns | 4320 |

### 120 名玩家

| 指标 | 改动前 p50 | 改动后 p50 | 倍数 |
| --- | --- | --- | --- |
| minecraft_playtime | 14.521 ms | 9.792 µs | 1483 |
| achievements | 51.436 ms | 8.125 µs | 6330 |
| distance | 14.610 ms | 8.000 µs | 1826 |
| mob_kills | 14.956 ms | 6.542 µs | 2286 |
| blocks_mined | 15.014 ms | 8.500 µs | 1766 |
| avg_move_distance | 29.232 ms | 9.083 µs | 3218 |
| login_days | 3.810 ms | 11.000 µs | 346 |
| join_streak | 3.759 ms | 9.166 µs | 410 |
| raw: minecraft:walk_one_cm | 14.939 ms | 7.708 µs | 1938 |

玩家数从 12 涨到 120，改动前的单次成本几乎线性放大 10 倍（`achievements` 5.16 ms 到 51.44 ms），改动后基本不变（0.375 µs 到 8.1 µs）。缓存命中后剩下的固定工作只有「拷贝 120 个结构体 + 排序」，所以延迟与数据规模脱钩。

`login_days` / `join_streak` 的基数较小（3.8 ms），因为它们不读玩家文件，只做一次全表扫描加 Go 侧的字符串切分；缓存同样把这次扫描从每个请求降为每 30 秒一次。

## 五、页面级聚合

把首页榜单会打的 9 个端点（8 个指标 + 1 个原始 stat）各请求 20 次，共 180 次：

| 语料 | 改动前总耗时 | 改动后总耗时 | 倍数 |
| --- | --- | --- | --- |
| 12 人 | 344.4 ms | 86.4 µs | 3986 |
| 120 人 | 3257.2 ms | 2040.3 µs | 1597 |

切换排序方向的成本也消失了。`achievements` 榜单已经缓存了降序结果，再请求升序时（120 人）：改动前 51.36 ms（完全重算），改动后 21.4 µs（拷贝 + 重排）。

## 六、缓存未命中的成本没有变差

缓存只影响命中路径，未命中路径理论上应与改动前相同。实测确实如此，甚至更好：

| 指标（120 人） | 改动前首次调用 | 改动后首次调用 |
| --- | --- | --- |
| minecraft_playtime | 27.22 ms | 19.63 ms |
| achievements | 80.51 ms | 65.30 ms |
| distance | 15.17 ms | 6.67 ms |
| mob_kills | 15.54 ms | 6.26 ms |
| blocks_mined | 15.88 ms | 6.69 ms |
| avg_move_distance | 28.90 ms | 12.82 ms |
| login_days | 3.70 ms | 3.19 ms |
| join_streak | 3.93 ms | 3.39 ms |
| raw: minecraft:walk_one_cm | 19.81 ms | 6.47 ms |

两点说明。第一行的 `minecraft_playtime` 是两个进程各自的第一个请求，包含一次性预热（页缓存、分配器、语言数据），对应「冷启动首个请求」，改动后 19.6 ms 比改动前 27.2 ms 更快。第二行之后，改动后之所以普遍降到四成左右，是因为数据层缓存已经在前面的指标里预热过：UUID 列表、白名单映射、玩家 stats 原始 JSON 都是跨指标共享的，所以第二个指标起就省掉了重复的文件读取。

进一步单独测量改动后的「纯未命中」路径（在 `leaderboard` 包内直接调用计算函数，进程全新启动）：

| 测量 | 第 1 次 | 后续（数据层已预热） |
| --- | --- | --- |
| distance | 30.91 ms | 7.06 ~ 7.71 ms |
| achievements | 89.98 ms | 34.6 ~ 123.1 µs |

对比同一语料下的改动前稳态（distance 14.61 ms、achievements 51.44 ms）可以看出：`distance` 这种只缓存原始 JSON 的指标，未命中时仍要为 120 个玩家各做一次 JSON 反序列化，约 7 ms；而 `achievements` 缓存的是算好的完成度结构体，未命中时几乎只剩循环开销。

## 七、并发突发

20 个客户端同时请求同一榜单（`achievements`，120 人，10 核）：

| 版本 | 三次实测墙钟时间 | 实际计算次数 |
| --- | --- | --- |
| 改动前 | 282.9 / 286.3 / 283.6 ms | 20 次 |
| 改动后 | 72.0 / 70.0 / 68.3 ms | 1 次 |

改动前每个请求都独立做完整工作，20 个请求在 10 核上排队跑，墙钟时间接近三倍单次耗时，CPU 消耗是单次的 20 倍。改动后 `singleflight` 把 20 个请求合并成一次计算（约 65 ~ 72 ms），其余 19 个直接命中缓存，墙钟时间等于一次计算。这一点对定时轮询、多人同时刷新页面这类场景很重要：延迟不再被请求数量放大。

单飞去重本身也有单元测试兜底：`internal/cache` 里断言 32 个并发调用只触发 1 次加载。

## 八、每个请求消失的工作量

以 120 人语料、`achievements` 榜单为例，单次请求的文件系统操作与 SQL：

| 操作 | 改动前 | 改动后（命中） |
| --- | --- | --- |
| `ReadDir` | 1 | 0 |
| 白名单读取 | 1 | 0 |
| stats 文件读取 | 0 | 0 |
| advancements 文件读取 | 120 | 0 |
| 大 JSON 反序列化 | 120 | 0 |
| SQL 查询（`user_preferences` / `users`） | 2 | 0 |

`minecraft_playtime` 是 122 次文件读取加 120 次解析，`avg_move_distance` 是 242 次读取加 240 次解析，改动后同为 0。

## 九、正确性边界

- 过期语义：TTL 是绝对过期（关闭了 `ttlcache` 的 touch-on-hit），不会因为读得多就一直不刷新。
- 失败不缓存：文件缺失、SQL 出错时直接返回错误，不写缓存。这一点有测试覆盖（先失败、后文件出现，第二次能读到）。
- 可见延迟：最坏情况下，远端文件更新后 30 秒内榜单仍显示旧数据；而统计数据本身在改动前就已经有 60 ~ 300 秒的同步延迟，因此增量很小。用户偏好（排行榜退出）的变化同样最多 30 秒生效；`disallow_public_game_stats` 的隐私判断不走缓存，仍然是即时的。
- 内存：榜单结果缓存容量 64，单个玩家数据缓存容量 1024 且带 LRU 淘汰，键受请求参数影响也被容量兜住。
- 不缓存 `ReadWhitelist`：绑定白名单需要最新文件，这一处刻意保持直读。

## 十、已知局限与后续可做

1. 基准是本机开发机、语料在页缓存里，改动前的数字其实是偏乐观的；真实服务器上 122 次冷读盘会更慢。
2. 数据库用 SQLite 而非生产的 MySQL，`login_days` 的绝对数值仅供参考，倍率关系（缓存命中 ≈ 常数）不受影响。
3. stats 只缓存了原始 JSON，未命中时仍要按指标重复反序列化（`distance` 约 7 ms）。若想进一步压低未命中成本，可以缓存解析后的结构体，或把统计结果物化成表，用 SQL 直接排序分页。
4. 目前是 TTL 失效，没有主动失效。文件同步完成后或用户偏好变更时主动清缓存，可以把 30 秒的可见延迟也去掉，代价是要把清缓存的调用点散到多个包。

## 十一、复现

基准测试文件没有进仓库（它需要放在包目录内才能调用包私有函数，直接放进仓库根会破坏 `go test ./...`，因此以 `.go.txt` 存在仓库根：`tmp-bench-leaderboard.go.txt`，含 `TestBench` 与 `TestConcurrentBurst`）。复现方式：

```bash
# 改动前的代码（本文对照版本）
git archive ce3f0c5 | tar -x -C /tmp/mc-before
# 当前工作树
rsync -a --exclude='.git/' --exclude='remote_data_cache/' --exclude='logs/' --exclude='*.db' ./ /tmp/mc-after/

# 两份代码使用同一份基准文件，并把 remote_data_cache 语料链到各自目录
cp tmp-bench-leaderboard.go.txt /tmp/mc-before/leaderboard/tmp_bench_test.go
cp tmp-bench-leaderboard.go.txt /tmp/mc-after/leaderboard/tmp_bench_test.go
ln -sfn /path/to/corpus /tmp/mc-before/remote_data_cache
ln -sfn /path/to/corpus /tmp/mc-after/remote_data_cache

cd /tmp/mc-before && go test ./leaderboard/ -run TestBench -v -count=1
cd /tmp/mc-after  && go test ./leaderboard/ -run TestBench -v -count=1
cd /tmp/mc-after  && go test ./leaderboard/ -run TestConcurrentBurst -v -count=1
```

完整基准文件见仓库根 `tmp-bench-leaderboard.go.txt`，核心结构如下：

```go
func TestBench(t *testing.T) {
	testutil.ChdirProjectRoot()
	store.C = store.Config{Driver: "sqlite", DBName: "bench", Path: "bench.db"}
	store.MustInitialize()
	store.AutoMigrate()
	defer os.Remove("bench.db")
	seedSamples(t, namesFromWhitelist()) // N 名玩家 x 30 天采样

	for _, metric := range benchMetrics { // 8 个指标 + 1 个原始 stat
		cold, _ := timeCall(func() error { _, err := BuildLeaderboard(Query{Metric: metric}); return err })
		samples := make([]time.Duration, 0, 20)
		for range 20 {
			d, _ := timeCall(func() error { _, err := BuildLeaderboard(Query{Metric: metric}); return err })
			samples = append(samples, d)
		}
		fmt.Printf("BENCH metric=%s cold=%s warm_p50=%s warm_max=%s\n",
			metric, cold, percentile(samples, 0.5), percentile(samples, 1))
	}
}
```

改动涉及的仓库文件：`internal/cache/{cache.go,cache_test.go}`、`leaderboard/leaderboard.go`、`playerdata/playerdata.go`，以及对应的两个测试文件；新增依赖 `github.com/jellydator/ttlcache/v3`，`golang.org/x/sync` 由间接依赖转为直接依赖。

# Go 服务指标分层与 Dashboard 设计

> 按【层级分类 - 面板设计 - 包含指标 - 面向场景】整理。手动触发方案保留，仅记录，不纳入持续采集。
> 本文关联 [登录性能专题中枢](design/2026-09-30-01-login-performance-hub.md)，指标编排见 [任务 46](backlog.md#46-文档代码与仪表盘编排一致)，测试采样见 [任务 55](backlog.md#55-测试期间采样与平台画像)。

## 一、服务健康

| 层级分类 | 面板设计 | 包含指标 | 面向场景 |
| --- | --- | --- | --- |
| 服务健康 | Metrics scrape status / Dependency status | `up`、`pgo_dependency_up{dependency}` | 错误率升高时先看依赖是否可达；`up=0` 先查抓取，`dependency_up=0` 先核对依赖，再继续查应用。 |

代码来源：[papp 健康指标](/root/code/pgo/pkg/papp/observability.go:28)、[健康检查](/root/code/pgo/pkg/papp/observability.go:81)；`up` 由 [Prometheus 抓取配置](../deploy/docker/config/prometheus.yml) 的抓取结果生成，应用不主动上报。依赖状态只在访问 `/healthz` 或 `/readyz` 时更新，`/metrics` 不执行健康检查；未初始化或尚未检查的依赖可能无数据，不能按健康处理。

## 二、请求 RED

| 层级分类 | 面板设计 | 包含指标 | 面向场景 |
| --- | --- | --- | --- |
| 请求 RED | Request rate / Error rate / p95 latency / Request rate by operation / Request p95 by operation | `pgo_http_requests_total{operation,result}`、`pgo_http_request_duration_seconds_bucket{operation,result,le}` 及 `_count`、`_sum` | 确认用户影响与拐点：P95 陡升、错误率从 0 变正。按 `operation` 拆分定位到具体接口。 |

代码来源：[papp 请求指标](/root/code/pgo/pkg/papp/observability.go:126)、[请求中间件](/root/code/pgo/pkg/papp/observability.go:139)，由 [Kratos 接线](/root/code/pgo/pkg/papp/kratosApp.go:101) 接入 HTTP/gRPC。`result` 为 `ok/error`，表示处理函数返回的错误；不等于所有 HTTP 状态码或 recovery 处理的 panic。请求面板使用 `irate(...[1m])`，基于最近两个样本；当前总览错误率分母最低取 1，低于 1 RPS 时不是严格比例，尚无错误序列时可能无数据。

## 三、依赖瓶颈

| 层级分类 | 面板设计 | 包含指标 | 面向场景 |
| --- | --- | --- | --- |
| 依赖瓶颈 | Database operation rate / Database error rate / Database p95 latency / Database connections / Connection wait rate / Average connection wait | `pgo_db_query_duration_seconds_count{operation,result}`、`_bucket{operation,result,le}`、`pgo_db_connections_open` / `in_use` / `idle`、`pgo_db_connection_wait_total`、`pgo_db_connection_wait_duration_seconds_total` | 请求 P95 升高时判断是数据库慢还是连接池排队。连接等待 > 0 且 `in_use` 接近连接上限时，优先排查连接池；数据库 P95 升高时，继续区分 SQL、连接等待与回调耗时。 |

代码来源：[pdb 操作耗时](/root/code/pgo/pkg/pdb/observability.go:16)、[连接池与等待指标](/root/code/pgo/pkg/pdb/observability.go:23)、[GORM 回调](/root/code/pgo/pkg/pdb/observability.go:50)。数据库 `operation` 是 `create/delete/query/raw/row/update`，与请求接口操作不同。耗时包含 GORM 操作的连接等待和回调，并非纯 SQL 执行时间；`in_use` 与 `open` 本身不代表配置的最大连接数。

## 四、进程资源

| 层级分类 | 面板设计 | 包含指标 | 面向场景 |
| --- | --- | --- | --- |
| 进程资源 | Process CPU / Process memory / Open file descriptors / File descriptor usage | `process_cpu_seconds_total`、`process_resident_memory_bytes`、`process_open_fds`、`process_max_fds` | 判断计算是否密集、内存是否增长、fd 是否泄漏。`process_cpu_seconds_total` 速率 ≈ 1 表示约一个核；fd 使用比例接近 1 时有耗尽风险。 |

代码来源：[papp 默认采集器入口](/root/code/pgo/pkg/papp/observability.go:203)，`process_*` 来自 client_golang 默认 [ProcessCollector](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/process_collector.go)，随 [默认注册表初始化](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/registry.go#L60) 注册。Process CPU 是进程累计 CPU 秒的速率，来自 Prometheus，与 pprof CPU 调用栈采样互补；RSS 与 Go heap 口径不同。

**默认隐藏**：`process_virtual_memory_bytes` 由基础采集器自动暴露，dashboard 不查询或展示。

## 五、Go runtime

| 层级分类 | 面板设计 | 包含指标 | 面向场景 |
| --- | --- | --- | --- |
| Go runtime | Go heap memory / Go allocation rate / Go heap objects / Goroutines / GC pause quantiles / GC cycle rate / Time since last GC | `go_memstats_heap_alloc_bytes`、`go_memstats_heap_sys_bytes`、`go_memstats_alloc_bytes_total`、`go_memstats_heap_objects`、`go_goroutines`、`go_gc_duration_seconds{quantile}`、`_count`、`go_memstats_last_gc_time_seconds` | 请求 P95 升高但 CPU 平、连接等待 0 时，看 GC 暂停与分配速率。`go_goroutines` 持续上升提示泄漏或阻塞堆积。 |

代码来源：[papp 默认采集器入口](/root/code/pgo/pkg/papp/observability.go:206)，`go_*` 来自 client_golang 默认 [GoCollector](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector_latest.go)。GC pause quantiles 是 runtime 当前保留 GC 样本的摘要分位值，不是所选 dashboard 时间窗口的 P95，也不是请求耗时。

**默认隐藏**：`go_memstats_heap_inuse_bytes`、`go_threads` 由基础采集器自动暴露，dashboard 不查询或展示。

## 六、调用栈画像（Pyroscope）

| 层级分类 | 面板设计 | 包含指标 | 面向场景 |
| --- | --- | --- | --- |
| 调用栈画像 | Pyroscope 火焰图 | CPU、heap、goroutine、block、mutex profile | 指标圈定窗口后，按同一时间范围查火焰图，定位到业务函数。Alloy 每 15s 拉取；block/mutex 仅测试期间采样，trace 按需获取。 |

代码来源：[pprof 路由](/root/code/pgo/pkg/papp/pprof.go:58) → [Alloy](../deploy/docker/config/config.alloy) → [Pyroscope 数据源](../deploy/docker/config/grafana/datasources/datasource.yml)。Alloy 每 15 秒发起采集，CPU profile 本身覆盖采样时段；heap profile 含存活与累计分配样本。直接打开 Pyroscope（`http://<部署主机>:24040`，见 [导航服务清单](../deploy/docker/portal/services.json)），按 `service_name="pgo-app"` 和与指标一致的时间范围查询 CPU、heap、goroutine、block、mutex 火焰图。Grafana 的 pgo-app dashboard 仅展示前五组指标。

采样控制与抓取分开：权限压测在数据准备完成后开启 block/mutex 采样会话，覆盖预热、正式读写及收尾，每档结束或取消后关闭；会话最长 24 小时，客户端设置测量时长加 5 分钟的到期时间。配置 `Diagnostics.BlockProfileRate`、`MutexProfileFraction` 为正数，缺失时测试直接报错，准备数据模式不启用采样。

- **控制**：`POST /debug/pprof/sampling?seconds=<有效期>` 返回会话 `id`，`DELETE /debug/pprof/sampling?id=<id>` 关闭该会话；并行会话与手动采集互斥，过期或服务停止时自动关闭。
- **抓取**：`GET /debug/pprof/goroutine` 是完整栈快照；`GET /debug/pprof/block?seconds=14` 和 `mutex?seconds=14` 使用 Go 标准区间差值。无 seconds 时默认 14 秒，最长 60 秒。关闭采样不会禁用抓取端点，不产生新样本时返回合法空增量；关停边界可能包含尚在完成的等待事件。
- **时间范围**：Alloy 每 15 秒抓取，增量区间约 14 秒，timeout 为 16 秒；分位数窗口与 profile 区间应对齐，间隔空隙可能漏掉事件。关闭采样不清空累计数据，区间差值避免重复上传历史。goroutine 无需采样开关，平时也会抓取。
- **分析边界**：block/mutex 反映 Go 同步等待和锁竞争，不代表数据库行锁；数据库驱动网络读取栈支持等待数据库响应，SQL 内部根因仍需数据库侧证据。runtime trace 继续使用 `go tool trace`，不写入 Pyroscope。

## 七、分析组合应用场景

以下组合用于缩小排查范围，不能单独确定根因。

- **请求 P95 ↑ + 数据库 P95 ↑** → 优先排查数据库访问，按数据库 `operation` 拆分；GORM 耗时也包含连接等待与回调。
- **请求 P95 ↑ + 数据库 P95 平 + 连接等待 > 0** → 优先排查连接池等待，结合 `in_use`、`open` 和配置的连接上限确认。
- **请求 P95 ↑ + CPU ↑ + 数据库平** → 优先排查应用计算，切 Pyroscope CPU profile 确认热点。
- **请求 P95 ↑ + CPU 平 + 连接等待 0 + GC 暂停 ↑** → 优先排查 GC 开销，看分配速率与 heap profile。
- **`go_goroutines` 持续 ↑ + P95 ↑** → 疑似泄漏或阻塞，在 Pyroscope 查看 goroutine 快照，结合 block/mutex 判断同步等待；网络等待可补充 runtime trace。
- **`process_open_fds` ↑ + 错误率 ↑** → 疑似 fd 堆积或泄漏，看使用比例并核对连接负载。
- **错误率 ↑ + `pgo_dependency_up=0`** → 最近一次健康检查提示依赖不可达，重新检查并排查依赖。

## 八、手动触发方案（保留）

以下诊断操作按需使用；goroutine 快照也供 Alloy 定时抓取：

- goroutine 快照：`GET /debug/pprof/goroutine`；原 POST runtime 接口仍可获取栈分布差值。
- block：`POST /debug/pprof/runtime?seconds=10&profile=block`
- mutex：`POST /debug/pprof/runtime?seconds=10&profile=mutex`
- runtime trace：`GET /debug/pprof/runtime-trace?seconds=5`

均需 `Diagnostics.Enabled=true` 且 `Diagnostics.Pprof=true`，block/mutex 需配置正采样率。POST 等待后直接返回二进制增量，最长 60 秒；原 POST goroutine 返回区间栈分布变化，GET goroutine 返回完整快照。测试采样会话与重叠 runtime 请求返回 409，未配置 block/mutex 采样率返回 503；请求完成、取消或服务停止时关闭对应采样。trace 最长 10 秒，使用 `go tool trace`，其余使用 `go tool pprof`。

## 九、文档、代码与 Dashboard 的一致性约定

- **顺序**：文档与代码按服务健康 → 请求 RED → 依赖瓶颈 → 进程资源 → Go runtime → 调用栈画像排列；Grafana dashboard 展示前五组指标，每组按本文「面板设计」从左到右、从上到下阅读。调用栈画像直接在 Pyroscope 查看。分析组合和手动触发作为使用说明，手动接口不加入 Alloy 持续采集。
- **名称**：本文前五组的英文面板标题与 [pgo-app dashboard](../deploy/docker/config/grafana/dashboards/pgo-app.json) 的 `title` 保持一致；分组中文名称与源码章节注释一致。画像保留文档与代码来源说明，不要求在 Grafana 增设同名分组；一致性包含职责与数据口径一致。
- **代码**：papp 按服务健康、请求 RED、依赖来源、进程与 Go 默认采集器入口排列；数据库实现仍在 pdb，画像与手动接口仍在 pprof，通过本文源码链接接续阅读。默认采集器提供的指标保留真实来源，不重复实现。
- **口径**：指标名称、标签、更新时机、查询含义与实际采集代码同步；面板中的曲线是默认展示，辅助暴露指标需在本组明确记录。Prometheus 为抓取序列添加 `job`、`instance` 标签。当前四个筛选器分别为数据源、实例、请求操作、数据库操作，两类操作筛选仅作用于各自面板。
- **维护**：新增、删除、重命名或调整指标与面板时，同一任务同步本文、源码章节/来源注释及 dashboard 的分组、标题、顺序、曲线和说明；涉及持续画像时同时核对 Alloy。以代码和配置核实事实，以本文组织阅读顺序。
- **验收**：检查正文前五组面板标题与 dashboard 顺序一致、来源链接有效、指标标签和查询匹配。开发期间做基本静态检查与编译；相关回归集中在任务收尾一次，实际页面加载留给部署后的展示验收。

# Grafana、Pyroscope 与 pprof 数据来源速查

> 核对日期：2026-10-03，以仓库当前代码和部署模板为准，不代表某台机器的实际运行状态。
> 关联中枢：[登录性能测试闭环](design/2026-09-30-01-login-performance-hub.md)。

## 1. 先找这几个位置

你提供数据给 Prometheus 的主要代码在两个文件中：

- [pkg/papp/observability.go](../pkg/papp/observability.go)：注册请求计数、请求耗时、依赖健康和业务状态指标；`requestObservabilityMiddleware` 更新请求数据；`newDiagnosticsServer` 用 `promhttp.Handler()` 暴露 `/metrics`。
- [pkg/papp/pprof.go](../pkg/papp/pprof.go)：独立负责 pprof 路由注册、CPU/heap、受控 runtime profile 与 trace；由诊断服务接线。
- [pkg/pdb/observability.go](../pkg/pdb/observability.go)：注册数据库操作耗时和连接池指标；GORM 回调记录操作耗时，连接池指标读取 `sql.DB.Stats()`。
- [pkg/papp/kratosApp.go](../pkg/papp/kratosApp.go)：`RunKratosApp` 为 HTTP/gRPC 挂载观测中间件，并在 `Diagnostics.Enabled` 为 true 时启动独立诊断服务。
- [deploy/docker/config/prometheus.yml](../deploy/docker/config/prometheus.yml)：Prometheus 每 15 秒抓取 `pgo-dr9:20002/metrics`，任务名为 `pgo-app`。应用没有主动把这些指标推送给 Prometheus。

三条数据链路分别是：

```text
请求中间件 / GORM 回调 / Go 默认采集器
  → 应用 :20002/metrics → Prometheus → Grafana 指标查询

Go runtime → 应用 :20002/debug/pprof/…
  → Alloy 定时抓取 CPU、heap → Pyroscope → Grafana profile 查询
  → 运维手动下载 profile → go tool pprof 本地分析

plogger 文件日志 → Alloy → Loki → Grafana 日志查询
```

Grafana 展示数据；Prometheus 保存指标时间序列；Pyroscope 保存按调用栈组织的 profile。pprof 是运行时数据接口及分析工具，这些 profile 不经过 Prometheus。

## 2. 指标、代码与 Dashboard 标题映射

仪表盘面板比 `observability.go` 中的指标定义多，主要有两处原因：

- **默认采集器**：项目引入 `client_golang` 后，其 [registry.go 的 init](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/registry.go#L60) 自动注册 `NewGoCollector()` 和 `NewProcessCollector()`，提供 `go_*` 与 `process_*`。项目只需在 `newDiagnosticsServer` 中挂载 `promhttp.Handler()`，就会一起暴露这些数据。
- **查询计算**：同一个原始指标可以支持多个面板。例如请求计数用于计算请求速率与错误率；Histogram 的桶用于计算 P95；连接等待的两个累计量用于计算平均等待时间。这些派生值在 [pgo-app.json](/root/code/pgo/deploy/docker/config/grafana/dashboards/pgo-app.json:23) 的 `panels[].targets[].expr` 中计算，没有对应的独立 Go 指标变量。

下面按仪表盘分组列出全部实际使用的原始指标。第一列记录**指标名及自身标签**，`job="pgo-app"` 和 `instance` 是 Prometheus 抓取时附加的公共标签，表中不逐行重复。Dashboard 标题与当前 JSON 一致。

### 健康状态

| 指标标签 | 指标描述 | 指标对应代码位置 | Dashboard 上的标题 |
| --- | --- | --- | --- |
| `up` | metrics 抓取成功为 1，失败为 0 | [Prometheus](/root/code/pgo/deploy/docker/config/prometheus.yml:23)，Prometheus 生成 | Metrics scrape status |
| `pgo_dependency_up`<br>`dependency` | 已初始化依赖可达为 1，失败为 0 | [papp](/root/code/pgo/pkg/papp/observability.go:43)：`healthHandler` | Dependency status |
| `pgo_business_status`<br>`name` | 业务主动设置的可用状态，1 或 0 | [papp](/root/code/pgo/pkg/papp/observability.go:91)：`SetBusinessStatus` | Business status |

`dependency` 默认为 `database、redis、rabbitmq`，也可由 `RegisterHealthCheck` 增加。依赖状态仅访问 `/healthz` 或 `/readyz` 时更新，抓取 `/metrics` 不触发检查；已有值可能来自上次检查。当前非测试业务代码没有调用 `SetBusinessStatus`，业务状态可能无数据。缺少序列显示 `No data`，不能解释为健康。

### 请求

| 指标标签 | 指标描述 | 指标对应代码位置 | Dashboard 上的标题 |
| --- | --- | --- | --- |
| `pgo_http_requests_total`<br>`operation、result` | 累计处理数，计算总速率、错误比例与分项速率 | [papp](/root/code/pgo/pkg/papp/observability.go:139)：`requestObservabilityMiddleware` | Request rate<br>Error rate<br>Request rate by operation |
| `pgo_http_request_duration_seconds_bucket`<br>`operation、result、le` | 累计耗时桶，估算总 P95 与分项 P95 | [papp](/root/code/pgo/pkg/papp/observability.go:140)：`requestDuration.Observe` | p95 latency<br>Request p95 by operation |

`operation` 是 Kratos 接口操作名，登录为 `/api.User/Login`，来自生成的 [userService_http.pb.go](../internal/pkg/api/userService_http.pb.go) 中的 `OperationUserLogin` 和 `http.SetOperation`，不是 `/user/token` URL。缺少操作名时为 `unknown`；`result` 按处理函数返回的 `err` 判断，取 `ok` 或 `error`。

**统计边界**：同一中间件挂在 HTTP 和 gRPC 上，两者都会计入 `pgo_http_*`，没有协议标签。记录的是经过中间件并正常返回的处理调用，未进入中间件的路由、解码失败或未返回到记录位置的 panic 不按普通返回路径计数；耗时不等于客户端端到端耗时。

`requestDuration` 注册的名称是 `pgo_http_request_duration_seconds`，Histogram 自动生成 `_bucket、_sum、_count` 三类系列。当前面板使用 `_bucket`，所以源码中找不到完整的 `_bucket` 名称定义。

### 数据库与连接池

| 指标标签 | 指标描述 | 指标对应代码位置 | Dashboard 上的标题 |
| --- | --- | --- | --- |
| `pgo_db_query_duration_seconds_count`<br>`operation、result` | GORM 操作累计次数，计算速率与错误比例 | [pdb](/root/code/pgo/pkg/pdb/observability.go:99)：`observeQueryEnd` | Database operation rate<br>Database error rate |
| `pgo_db_query_duration_seconds_bucket`<br>`operation、result、le` | GORM 操作耗时累计桶，估算 P95 | [pdb](/root/code/pgo/pkg/pdb/observability.go:99)：`observeQueryEnd` | Database p95 latency |
| `pgo_db_connections_open` | 当前已建立连接数 | [pdb](/root/code/pgo/pkg/pdb/observability.go:24)：`getDBStats().OpenConnections` | Database connections |
| `pgo_db_connections_in_use` | 当前使用中的连接数 | [pdb](/root/code/pgo/pkg/pdb/observability.go:28)：`getDBStats().InUse` | Database connections |
| `pgo_db_connections_idle` | 当前空闲连接数 | [pdb](/root/code/pgo/pkg/pdb/observability.go:32)：`getDBStats().Idle` | Database connections |
| `pgo_db_connection_wait_total` | 累计等待次数，计算等待速率与均值分母 | [pdb](/root/code/pgo/pkg/pdb/observability.go:36)：`getDBStats().WaitCount` | Connection wait rate<br>Average connection wait |
| `pgo_db_connection_wait_duration_seconds_total` | 累计等待秒数，计算平均等待时间的分子 | [pdb](/root/code/pgo/pkg/pdb/observability.go:40)：`getDBStats().WaitDuration` | Average connection wait |

`operation` 为 `create、delete、query、raw、row、update`，`result` 按 `db.Error` 判断。`queryDuration` 注册的名称是 `pgo_db_query_duration_seconds`，`_count` 与 `_bucket` 由 Histogram 自动生成。

[mysql.go](../pkg/pdb/mysql.go)、[sqlite.go](../pkg/pdb/sqlite.go)、[pgsql.go](../pkg/pdb/pgsql.go) 初始化 GORM 时调用 `registerQueryObservability`，在回调前后用 `observeQueryStart` 和 `observeQueryEnd` 计时。统计的是 GORM 操作回调耗时，不是数据库服务端纯 SQL 执行时间，没有 SQL 文本或表名标签。

连接池指标在抓取时读取当前默认数据库的 `sql.DB.Stats()`，未取得数据库时返回零值；不是 MySQL 服务器全局连接数。`open` 包含 `in_use` 与 `idle`，三条曲线不相加。

请求与数据库 Histogram 都采用 `prometheus.DefBuckets`：`0.005、0.01、0.025、0.05、0.1、0.25、0.5、1、2.5、5、10` 秒，另有 `+Inf`。`le` 为桶上界；P95 按桶估算。

### 进程资源，默认 Process Collector

本组没有项目自定义采集函数。[process_collector.go](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/process_collector.go) 定义指标，[process_collector_procfsenabled.go](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/process_collector_procfsenabled.go#L28) 的 `processCollect` 在 Linux 等支持的平台读取 `/proc` 中的进程数据。

| 指标标签 | 指标描述 | 指标对应代码位置 | Dashboard 上的标题 |
| --- | --- | --- | --- |
| `process_cpu_seconds_total` | 累计 CPU 秒数，取速率后为 CPU 秒/秒 | [processCollect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/process_collector_procfsenabled.go#L28)：`stat.CPUTime()` | Process CPU |
| `process_resident_memory_bytes` | 进程驻留内存 RSS，字节 | [processCollect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/process_collector_procfsenabled.go#L28)：`stat.ResidentMemory()` | Process memory |
| `process_virtual_memory_bytes` | 进程虚拟内存，字节 | [processCollect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/process_collector_procfsenabled.go#L28)：`stat.VirtualMemory()` | Process memory |
| `process_open_fds` | 当前打开的文件描述符数量 | [processCollect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/process_collector_procfsenabled.go#L28)：`FileDescriptorsLen()` | Open file descriptors<br>File descriptor usage |
| `process_max_fds` | 文件描述符上限 | [processCollect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/process_collector_procfsenabled.go#L28)：`Limits().OpenFiles` | Open file descriptors<br>File descriptor usage |

CPU 秒/秒为 1 约等于一个逻辑 CPU 核，未按主机或容器配额归一化，允许大于 1。RSS 与 Go 堆分配量口径不同，文件描述符采集依赖平台支持。

### Go runtime，默认 Go Collector

本组也没有项目自定义采集函数。[go_collector_latest.go](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector_latest.go#L306) 的 `goCollector.Collect` 调用 `runtime/metrics.Read` 并转换内存统计；[go_collector.go](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L244) 的 `baseGoCollector.Collect` 提供 goroutine、线程和 GC 基础指标。源码链接固定为本项目当前依赖的 `client_golang v1.23.2`。

| 指标标签 | 指标描述 | 指标对应代码位置 | Dashboard 上的标题 |
| --- | --- | --- | --- |
| `go_memstats_heap_alloc_bytes` | 当前堆对象分配字节数 | [goRuntimeMemStats](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L26)：`HeapAlloc` | Go heap memory |
| `go_memstats_heap_inuse_bytes` | 使用中的堆 span 字节数 | [goRuntimeMemStats](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L26)：`HeapInuse` | Go heap memory |
| `go_memstats_heap_sys_bytes` | 从系统取得的堆空间字节数 | [goRuntimeMemStats](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L26)：`HeapSys` | Go heap memory |
| `go_memstats_alloc_bytes_total` | 累计分配字节数，取速率后为分配字节/秒 | [goRuntimeMemStats](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L26)：`TotalAlloc` | Go allocation rate |
| `go_memstats_heap_objects` | 当前堆对象数 | [goRuntimeMemStats](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L26)：`HeapObjects` | Go heap objects |
| `go_goroutines` | 当前 goroutine 数 | [baseGoCollector.Collect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L244)：`runtime.NumGoroutine()` | Goroutines and threads |
| `go_threads` | runtime 创建的 OS 线程数 | [baseGoCollector.Collect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L244)：`getRuntimeNumThreads()` | Goroutines and threads |
| `go_gc_duration_seconds`<br>`quantile` | GC 暂停时间分位值，单位秒 | [baseGoCollector.Collect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L244)：`debug.ReadGCStats()` | GC pause quantiles |
| `go_gc_duration_seconds_count` | 累计 GC 次数，取速率后为 GC 次/秒 | [baseGoCollector.Collect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L244)：`stats.NumGC` | GC cycle rate |
| `go_memstats_last_gc_time_seconds` | 最近一次 GC 的 Unix 时间，秒 | [baseGoCollector.Collect](https://github.com/prometheus/client_golang/blob/v1.23.2/prometheus/go_collector.go#L244)：`stats.LastGC` | Time since last GC |

`goRuntimeMemStats` 定义兼容 `runtime.MemStats` 的指标映射；当前 Go 版本实际由 `runtime/metrics` 填充这些字段，不是项目直接调用 `runtime.ReadMemStats()`。GC 暂停 summary 展示 `quantile=0.5、0.75、1`，分别为 P50、P75 和最大值；`_count` 是同一个 summary 的累计次数。

### Dashboard 派生值怎么查代码

这些值都在 [pgo-app.json](/root/code/pgo/deploy/docker/config/grafana/dashboards/pgo-app.json:23) 的查询表达式中计算：

- **速率**：请求、数据库操作、连接等待、CPU、内存分配和 GC 次数使用 `irate(...[1m])`，取最近两个抓取点的增量速率。
- **P95**：请求和数据库耗时用 `histogram_quantile(0.95, ...)` 从 `_bucket` 估算。
- **错误率**：错误结果的次数增量速率除以总次数增量速率；请求保留分母至少为 1 的口径，数据库在有操作但无错误序列时显示 0。
- **平均连接等待**：等待秒数增量速率 / 等待次数增量速率；没有等待时无有效均值。
- **文件描述符使用比例**：`process_open_fds / process_max_fds`。
- **距上次 GC 的时间**：`time() - go_memstats_last_gc_time_seconds`；没有发生过 GC 时无数据。

GC 暂停分位数由 Go collector 提供，不在 Dashboard 中用 Histogram 计算，也不代表当前选定时间范围的精确分位数。完整原始指标仍以实际 `/metrics` 为准；上述表格聚焦当前 25 个面板，未列出其他已暴露但未使用的系列。

## 3. Grafana 当前展示哪些数据

### 预置应用仪表盘

[pgo-app.json](../deploy/docker/config/grafana/dashboards/pgo-app.json) 中的 `PGO Application` 当前有 5 个分组、25 个数据面板。顶部提供 Prometheus 数据源、实例、请求操作和数据库操作筛选，默认选择全部实例与操作。多选通过正则匹配指标标签，变量行为见 [Grafana 变量说明](https://grafana.com/docs/grafana/latest/visualizations/dashboards/variables/add-template-variables/)。

- 健康状态：metrics 抓取状态、依赖健康、业务状态。
- 请求：总请求速率、P95、错误率，以及按实例与操作拆分的请求速率、P95。
- 数据库与连接池：按操作及结果拆分的速率、操作 P95、操作错误率、连接数、等待速率和平均等待时间。
- 进程资源：CPU、RSS 与虚拟内存、文件描述符数量与使用比例。
- Go runtime：堆内存、分配速率、堆对象、goroutine 与线程、GC 暂停分位数、GC 频率和距上次 GC 的时间。

实例筛选作用于全部面板；请求操作只影响请求面板，数据库操作只影响数据库操作面板。连接池和运行时数据属于实例整体，不随操作筛选变化。

原有三个请求面板保留 `irate(...[1m])` 口径，并限定 `job="pgo-app"` 和所选实例、操作。实际查询如下：

**Request rate，请求速率，单位次/秒：**

```promql
sum(irate(pgo_http_requests_total{job="pgo-app",instance=~"${instance:regex}",operation=~"${operation:regex}"}[1m]))
```

**p95 latency，服务处理耗时 P95，查询结果单位秒：**

```promql
histogram_quantile(0.95, sum(irate(pgo_http_request_duration_seconds_bucket{job="pgo-app",instance=~"${instance:regex}",operation=~"${operation:regex}"}[1m])) by (le))
```

**Error rate，处理错误比例，结果为比例值：**

```promql
sum(irate(pgo_http_requests_total{job="pgo-app",instance=~"${instance:regex}",operation=~"${operation:regex}",result="error"}[1m])) / clamp_min(sum(irate(pgo_http_requests_total{job="pgo-app",instance=~"${instance:regex}",operation=~"${operation:regex}"}[1m])), 1)
```

三个总览面板聚合当前选中的实例与操作；分项面板保留实例、操作及适用的结果标签。`irate` 使用窗口内最近两个采样点，当前抓取间隔为 15 秒；`[1m]` 不是一分钟平均值，见 [Prometheus irate 说明](https://prometheus.io/docs/prometheus/latest/querying/functions/#irate)。错误率分母最低取 1，低于 1 RPS 时不等于严格的错误请求占比，缺少错误序列时也可能显示无数据。

新增面板已覆盖数据库、连接池、依赖健康、业务状态和常用 Go runtime 指标。状态面板使用当前瞬时值，缺失序列显示 `No data`，不补成健康；依赖值仍只在调用健康接口时更新，当前业务状态可能无数据。

数据库平均连接等待时间使用等待秒数增量除以等待次数增量，无等待时没有有效均值；数据库错误率在有操作但无错误序列时显示零，无操作时没有有效比例。GC 暂停分位数按实例读取 summary，不跨实例聚合，也不是面板所选时间范围的精确分位数。

也可在 Grafana Explore 选择 Prometheus 查询，例如：

```promql
# 登录速率
sum(irate(pgo_http_requests_total{job="pgo-app",operation="/api.User/Login"}[1m]))

# GORM 各类操作 P95
histogram_quantile(0.95, sum(rate(pgo_db_query_duration_seconds_bucket{job="pgo-app"}[5m])) by (le, operation))

# 应用连接池使用量
pgo_db_connections_in_use{job="pgo-app"}

# 应用进程每秒 CPU 消耗，1 表示约一个 CPU 核的计算量
rate(process_cpu_seconds_total{job="pgo-app"}[1m])
```

### 主机、容器、PM2 和日志

这些数据也能在 Grafana 中查看，但来源不在业务 Go 指标代码中：

- [host.json](../deploy/docker/config/grafana/dashboards/host.json)：主机 CPU、内存、磁盘、网络等，Prometheus 抓取 `pgo-node-exporter:9100`，job 为 `host`，主要为 `node_*` 指标。
- [docker.json](../deploy/docker/config/grafana/dashboards/docker.json)：容器 CPU、内存、网络、文件系统等，抓取 `pgo-cadvisor:8080`，job 为 `docker`，主要为 `container_*` 指标。
- [pm2.json](../deploy/docker/config/grafana/dashboards/pm2.json)：PM2 进程状态及资源数据，抓取 `pgo-dr9:9988` 的 `pm2-prom-module`；是否有数据取决于部署端模块是否运行。
- Loki 日志：由 [pkg/plogger/init.go](../pkg/plogger/init.go) 和 [zaplog.go](../pkg/plogger/zaplog.go) 提供文件日志；Compose 将 `backend/logs` 挂到 Alloy 的 `/opt/app_logs`，Alloy 匹配 `*.log` 后写入 Loki。
  - [config.alloy](../deploy/docker/config/config.alloy) 提取 `T、L、M、caller、tid` 字段，将时间戳用于日志时间，消息 `M` 用作输出内容。
  - 当前提升到标签的是 `app、date`，另有固定 `env="prod"、job="pgo"`；`tid、caller、level` 虽被解析，但没有配置为标签或结构化元数据，不应假设能直接按这些字段查询。
  - Explore 中可用 `{job="pgo"}` 查询应用日志。日志读取依赖实际写入被挂载的文件。

Redis/MySQL exporter 抓取配置目前被注释，不能把连接池指标理解成已有完整 Redis/MySQL 服务器监控。

## 4. Pyroscope 持续保存什么

来源：[pprof](/root/code/pgo/pkg/papp/pprof.go:53) 注册 pprof 路由，由 `newDiagnosticsServer` 挂载到诊断服务，采集规则见 [config.alloy](../deploy/docker/config/config.alloy)。应用没有引入 Pyroscope SDK 主动推送。

- 目标：`pgo-dr9:20002`，标签 `service_name="pgo-app"`。
- 周期：`scrape_interval="15s"`，超时为 `14s`。
- CPU：`profile.process_cpu.enabled=true`，通过 `/debug/pprof/profile` 获取执行采样调用栈，用于看计算热点。
- heap：`profile.memory.enabled=true`，路径显式设置为 `/debug/pprof/heap`，用于看内存分配调用栈。
- 去向：`pyroscope.write.local` 写入 `http://pgo-pyroscope:4040`；[Grafana 数据源](../deploy/docker/config/grafana/datasources/datasource.yml) 已配置 Pyroscope，可在 Explore/对应插件中按服务与时间范围选择 profile。
- goroutine、block、mutex 的 Alloy 采集均为 false；runtime trace 也没有持续采集。

heap profile 含 `inuse_space、inuse_objects、alloc_space、alloc_objects` 四种样本维度，分别对应存活字节、存活对象数、累计分配字节、累计分配对象数。Pyroscope 界面的具体 profile 类型名称以实际摄入数据为准。CPU profile 展示采样到的执行开销，不能据此直接还原完整请求耗时。[Go pprof 官方说明](https://pkg.go.dev/runtime/pprof#Profile)

当前没有为 profile 设置登录接口或请求 ID 标签；`service_name` 也不是每个业务接口的名称。定位登录热点需要结合压测时间范围和调用栈中的函数。

## 5. 手动 pprof 和 trace 能看到什么

接口均由 [pprof](/root/code/pgo/pkg/papp/pprof.go:53) 提供，需要 `Diagnostics.Enabled=true` 且 `Diagnostics.Pprof=true`。标准库支持的 profile 不代表本项目全部开放：当前没有挂载 `/debug/pprof/` 索引、`/allocs`、`/threadcreate` 或标准 `/trace` 路径。

- **CPU**：`GET /debug/pprof/profile?seconds=10`，采集该时段 CPU 执行调用栈，用 `top、list、调用图` 定位热点。`cpuProfileHandler` 用互斥锁串行化 CPU 采集；手动采集可能与 Alloy 争用，不是两套独立采样。
- **heap**：`GET /debug/pprof/heap`，获取分配调用栈，可按上述四个维度分析；默认看 `inuse_space`。存活数据通常反映最近完成的 GC，`?gc=1` 可请求先执行 GC；heap 不等于进程 RSS。
- **goroutine**：`GET /debug/pprof/goroutine`，查看当前 goroutine 调用栈及数量分布；`?debug=2` 返回便于阅读的文本堆栈，需要先开启诊断窗口。
- **block**：`GET /debug/pprof/block`，看同步原语导致的阻塞位置，例如 channel、锁、WaitGroup；需要先开启窗口并配置正的 `BlockProfileRate`。
- **mutex**：`GET /debug/pprof/mutex`，看锁竞争开销，调用栈通常指向持锁方释放锁的位置；需要先开启窗口并配置正的 `MutexProfileFraction`。
- **runtime trace**：`GET /debug/pprof/runtime-trace?seconds=5`，查看调度、goroutine 状态变化、GC 等时间线，最长 10 秒。文件用 `go tool trace` 打开，不使用 `go tool pprof`。

heap、block、mutex 是采样或累计数据，block/mutex 开窗不会清空以前的统计；严格比较两个时段时需比较前后 profile。各类 profile 的语义见 [Go runtime/pprof](https://pkg.go.dev/runtime/pprof#Profile)。

### 配置和诊断窗口

[configs/common.yaml](../configs/common.yaml) 模板开启诊断，监听 `0.0.0.0:20002`；代码的地址缺省值为 `127.0.0.1:19090`，实际以加载配置为准。

- CPU 使用标准 `net/http/pprof` 默认的 100 Hz 采样。
- `MemProfileRate: 0` 表示本项目不覆盖 runtime 默认值，不表示关闭内存采样。
- 模板 `BlockProfileRate: 1、MutexProfileFraction: 1`，只在开启窗口期间应用，窗口到期将运行时参数恢复为 0。
- `POST /debug/pprof/runtime?seconds=30&profiles=goroutine,block,mutex` 开启最长 60 秒的访问窗口；已有窗口时再次开启返回 409，未激活的 profile 返回 403。
- 配置 `Pprof=false` 时仍保留 `/metrics、/healthz、/readyz`。

### 常用下载与分析命令

以下以可访问的本机诊断端口为例；`curl` 在前台执行，命令结束即退出：

```bash
# 查看原始应用指标
curl -fsS --max-time 10 http://127.0.0.1:20002/metrics

# 触发健康检查，同时刷新依赖健康指标
curl -fsS --max-time 10 http://127.0.0.1:20002/readyz

# 下载 heap 并查看存活内存 / 累计分配热点
curl -fsS --max-time 10 http://127.0.0.1:20002/debug/pprof/heap -o /tmp/pgo-heap.pprof
go tool pprof -top -inuse_space /tmp/pgo-heap.pprof
go tool pprof -top -alloc_space /tmp/pgo-heap.pprof

# 下载 10 秒 CPU profile 并查看热点
curl -fsS --max-time 30 'http://127.0.0.1:20002/debug/pprof/profile?seconds=10' -o /tmp/pgo-cpu.pprof
go tool pprof -top /tmp/pgo-cpu.pprof

# 开启 block/mutex 窗口，在窗口内施加负载，等待 10 秒后下载
curl -fsS --max-time 10 -X POST 'http://127.0.0.1:20002/debug/pprof/runtime?seconds=30&profiles=block,mutex'
sleep 10
curl -fsS --max-time 10 http://127.0.0.1:20002/debug/pprof/block -o /tmp/pgo-block.pprof
curl -fsS --max-time 10 http://127.0.0.1:20002/debug/pprof/mutex -o /tmp/pgo-mutex.pprof
go tool pprof -top /tmp/pgo-block.pprof
go tool pprof -top /tmp/pgo-mutex.pprof

# trace 的输出格式不同
curl -fsS --max-time 15 'http://127.0.0.1:20002/debug/pprof/runtime-trace?seconds=5' -o /tmp/pgo-runtime.trace
go tool trace /tmp/pgo-runtime.trace
```

`go tool trace` 会以前台方式提供查看服务，结束时按 Ctrl+C。需要交互调用图时可执行 `go tool pprof -http=127.0.0.1:8081 /tmp/pgo-cpu.pprof`，同样按 Ctrl+C 停止。

[diagnostics.go](../cmd/pgo/tools/diagnostics/diagnostics.go) 也提供 `health、metrics-url、profile` 命令，在 pgo 的工具分组中注册。注意当前实现：

- `metrics-url` 只打印 URL，不读取指标。
- `profile` 支持 `cpu、heap、goroutine、block、mutex、trace`；block/mutex/goroutine 开窗后立即 GET，`--seconds` 是开放窗口而不是等待采样时长。
- trace 必须显式指定不超过 10 的 `--seconds`，因为命令默认值为 30；trace 下载后应自行运行 `go tool trace`。当前 `--open` 对所有类型都调用 pprof，不适合 trace。

## 6. 入口、告警和快速定位

- 导航页：`http://<部署主机>:20080`，端口清单在 [services.json](../deploy/docker/portal/services.json)。
- Prometheus：`:29090`；Grafana：`:23000`；Pyroscope：`:24040`。
- 应用诊断：`:20002`。Compose 当前通过 `20000-20010` 范围映射包含此端口，访问范围应由部署网络控制；模板要求诊断端口不暴露公网。
- [pgo-app.yml](../deploy/docker/config/prometheus-rules/pgo-app.yml) 定义三条告警：metrics 抓取失败持续 2 分钟；五分钟速率口径下错误率超过 5% 持续 5 分钟；依赖指标为 0 持续 2 分钟。依赖告警受前述健康检查更新时机限制。

忘记数据从哪里来时，按下面顺序查：

1. 看应用 `/metrics` 是否有目标指标，判断应用是否已注册或产生带标签的数据。
2. 看 Prometheus 的 `pgo-app` target 和 `up`，核对抓取地址是否匹配实际诊断监听地址。
3. 看 Grafana 查询是否选对数据源、时间范围与标签，区分当前面板与未建面板的指标。
4. profile 则查 Alloy 的目标、采集配置与写入日志，再在 Pyroscope 按 `service_name="pgo-app"` 和同一时间范围查看。

以登录为例：[userService.go](../internal/userService/userService.go) 初始化数据库并调用 `RunKratosApp`；[UserServer.Login](../internal/userService/service/user.go) 调用 [UserDAO.GetOrAdd](../internal/userService/data/dao_User.go)，GORM 回调记录数据库操作，中间件记录整个处理调用。函数自身没有调用 Prometheus API，指标来自公共接线；CPU/heap 调用栈来自 Go runtime。

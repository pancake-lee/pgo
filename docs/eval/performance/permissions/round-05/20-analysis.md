# round-05 正式负载监控分析

> 专题入口：[性能测试闭环中枢](../../../../design/2026-09-30-01-login-performance-hub.md)。关联任务：[73](../../../../archive/v0.0.12.md#73-round-05-平台数据与硬件瓶颈分析)。

本次窗口为北京时间 **2026-10-09 11:46:00–11:47:00**，对应 UTC 03:46:00–03:47:00，Unix 时间 1791517560–1791517620。直接读取部署环境 Grafana 的仪表盘与 Prometheus 数据源代理，以及 Pyroscope 公共查询 API。

**结论：没有观察到主机全部 CPU、单个逻辑核、内存容量、磁盘或物理网卡持续满载。已经观察到 CPU 调度等待与数据库连接池等待；应用查询对象构造、结果扫描和内存分配存在明确成本。这些证据支持数据库访问链路与共享 CPU 竞争共同放大尾延迟，尚不能把它归结为某个硬件的硬上限。**

后续 [任务 74](../../../../archive/v0.0.12.md#74-复用-gorm-gen-query-消除重复构造) 已实现默认 Query 复用，本地 GetQuery 分配降为零；用户已完成 round-06/07 auto 对照，实测和本人理解见[综合复盘](../README.md)。本文 profile 与压测数字仍描述修复前的 round-05，不能当作修复后的实测收益。

## 数据与时间口径

- 用户后补的[分配火焰图](p-alloc-space-1.png)、[分配函数表](p-alloc-space-2.png)、[CPU 火焰图](p-cpu.png)选中 11:46–11:48，比本报告正式分钟宽。函数表 Use Total 为 1.19 GiB；CPU 提示框 650ms/2.01% 对应一个 GetQuery 调用位置，不能替代本文三个分支合计的函数占比。
- [Grafana 原始数据](21-grafana-data.json)：线上 Host Monitor、PGO Application、Docker Monitor 的面板定义；相关 Host/Application 面板查询结果，以及逐核 CPU、容器 CPU、原始计数器、cAdvisor 宿主机网络等补充查询。
- [Pyroscope 原始数据](22-pyroscope-data.json)：CPU、分配、存活内存、goroutine、block、mutex 的公共 API pprof JSON 响应，包含函数与样本栈；CPU/分配/block/mutex 的 1 秒查询步长时间序列。
- Grafana 查询范围为上述一分钟，步长 15 秒。Host 面板的 `$__rate_interval` 固定为 1 分钟；应用面板沿用线上定义的 `irate(...[1m])`。原始 counter 补查 90 秒，仅用于核对边界和相邻样本，不能当作正式负载延长。
- 主机实际抓取约在每分钟 :13/:28/:43/:58，应用约在 :10/:25/:40/:55。11:46:00 的面板点使用此前抓取数据，**不是正式负载样本**；`rate[1m]` 开头仍混入预热与停发阶段。本文不把五个面板点均值当作整分钟精确均值。
- Pyroscope CPU 样本时间落在约 11:46:07/:22/:37/:52。它们是周期采样数据，窗口边界可能覆盖相邻阶段。1 秒查询步长用于避免时间桶扩展，并不意味着每秒采集一次。15 秒步长的 SelectSeries 会包含边界桶，不能直接求和替代合并 profile。
- 15 秒抓取不能排除更短的 CPU/连接数峰值。本文“峰值”均指可见采样峰值。

## 本次压测结果

本次目录中的报告已经替换旧轮次数字，不能继续引用“1 次 HTTP 500”。

- 热点组：6000/6000 成功，平均 46.754ms，P50 8.234ms，P95 303.892ms，P99 524.883ms，最大 1.061s。
- 对照组：6000/6000 成功，平均 48.205ms，P50 7.884ms，P95 325.055ms，P99 550.404ms，最大 1.1s。
- 写入：296 次全部成功，实际吞吐 4.91/s，P95 149.442ms，P99 287.756ms。

两组读数接近，尾延迟明显高于中位数；当前 200 RPS 窗口成功，但不能据此确定更高负载的零错误容量边界。

## HostMonitor：硬件有没有到顶

- **CPU**：4 个逻辑核，Go 的 GOMAXPROCS 也为 4。CPU Busy 的一分钟平滑值在面板范围内为 62.79%–74.49%；CPU Basic 的相邻抓取间隔总忙碌率最高约 78.53%，逐核最高约 79.4%。各核相近，未出现某一个逻辑核接近 100%、其余核闲置的采样证据。末段仍约有 21% CPU idle。
- **CPU 调度压力**：load1 从 3.47 升至 5.15，超过 4 核；末段 running processes 快照为 23。每核 schedstat 等待/运行合计的比例约 52%–67%，说明可运行任务已有排队。这个比例衡量累计任务等待时间，不是 CPU 利用率，也不表示“67% CPU 已耗尽”。即使平均利用率没有到 100%，突发并发和共享任务仍可能造成延迟。
- **CPU 时间构成**：末段 user 约 51.95%、system 20.15%、irq+softirq 3.90%，iowait 约 0.05%，steal 为 0。CPU 压力较明显，尚无全核持续耗尽的证据。
- **内存**：总量约 11.32 GiB，正式范围内可用内存最低约 4.31 GiB。约 2.79 GiB swap 占用基本不变，换页速率最高不足 0.5 页/秒；OOM kill 增量为 0。不能将已经使用 swap 直接解释为本次压测内存不足。
- **磁盘**：sda 忙碌率最高约 1.39%，写入最高约 1.67 MiB/s、141.47 IOPS；平均队列长度最高约 0.036，平均写入耗时约 0.15–0.27ms。结合极低 iowait，未见块设备整体吞吐/队列饱和。平均值不能排除少量同步写的尖峰，但不支持磁盘整体到顶。
- **网络**：HostMonitor 的流量查询仅返回 exporter 容器 `eth0`，而速度指标包含宿主机 `enp4s0f2`、网桥、veth；Network Saturation 面板无匹配结果。部署配置使用 bridge 网络，和这一命名空间不一致的现象相符。**该 HostMonitor 网络面板不能用于证明宿主机网卡是否满载。**补充 cAdvisor `id="/"` 数据显示，物理接口 enp4s0f2 接收最高约 39.9 KB/s、发送约 10.6 KB/s，速度指标为 1 Gbit/s；可见流量远低于标称带宽。业务主要流经 Docker 网桥，相关网桥流量约 1 MB/s，也没有带宽耗尽的迹象。
- **监控缺口**：pressure collector 的 success 为 0，PSI 面板没有数据；容器 CPU quota 与 throttling 查询没有数据，不能将空结果解释为“零限流”。缺少线程级 MySQL CPU/profile，不能排除数据库内部某条执行路径或短时单线程瓶颈。

## 应用与数据库链路

应用 Process CPU 在面板上约为 **0.61–0.89 个核心**，末段占 4 核主机容量约 22.3%。MySQL 容器可见峰值约 0.77 核，应用容器约 0.89 核。另一个 `dr9` 容器约 0.29–0.37 核，RabbitMQ 约 0.20–0.29 核，cAdvisor 在该范围约 0.12–0.34 核。主机 CPU 同时承担这些工作，不能把全部宿主机 CPU 都算到权限请求上。

正式稳定段读请求约 200/s，数据库 query 约 600/s，与权限接口串行执行三次查询一致。窗口末段权限请求直方图 P95 约 449ms、单次 GORM query P95 约 166ms；这是最后相邻抓取区间的估计值，与 Vegeta 整分钟 P95 约 304/325ms 不同。GORM 指标覆盖 callback 包围的过程，包括取连接、执行和结果处理，并不等于 MySQL 服务端纯 SQL 执行时间。

连接池末段出现 **856 次新增等待**，累计等待 **13.796 秒**，每次平均约 **16.1ms**，对应最后约 15 秒区间的 57.1 次等待/秒。此前相邻采样计数没有增长，等待集中在约 11:46:40.429–11:46:55.431。采样时 open 最大 48、in_use 最大 44，并不排除两次抓取之间曾经触及连接上限。运行配置未保存部署时的实际池上限，不能把代码默认 64 写成已验证的生产配置。

因此可以确认存在**连接池容量等待**，但不能将其直接等同于 MySQL 最大连接数不足或硬件耗尽。数据库服务/结果处理变慢使连接占用变长，也会增加池等待。应用 goroutine 从正式范围内的 112 上升至 262，符合并发在途工作增加；进程 RSS 峰值约 71.4 MiB、堆 alloc 约 24.9 MiB，并无应用内存容量耗尽迹象。

## Pyroscope：哪些成本可以定位到代码

合并 CPU profile 为 **42.96 CPU 秒**，分配 profile 约 **3.72 GB**。以下百分比均为同一类型 profile 的累计样本占比，父子函数存在包含关系，**不能相加**；CPU profile 只度量执行 CPU 的时间，不能直接度量数据库响应等待。

- **结果扫描**：`gorm.Scan` 占应用 CPU 样本约 34.94%，其中 `database/sql.Rows.Scan` 约 13.18%。权限列表查询 `GetByRoleIDs` 累计约 38.36% CPU、50.13% 分配，是当前查询链路中成本最高的业务分支。
- **查询对象重复构造**：`db.GetQuery → query.Use` 占 CPU 约 **6.89%**、分配约 **34.51%**，对应约 **1.28 GB** 分配。`GetQuery()` 每次都调用 `query.Use()`，后者创建全部 **13 个表**的查询对象；三个权限 DAO 分别调用一次。分配栈中出现本次权限查询不使用的 CourseSwapRequest、Task 等对象，和源码机制一致。这是已定位的软件额外成本，不能把这些初始化开销解释为“权限数据太多，硬件只能这样”。
- **GC**：后台 `gcBgMarkWorker` 占 CPU 约 11.43%，稳态分配速率约 66 MiB/s、GC 约 7–8 次/秒。GC 是实际 CPU 成本，但短暂停顿中位数约 0.12–0.14ms；summary 的最大值约 19–26ms 来自滚动历史，不能逐项归因到本分钟，更不足以单独解释约 0.5 秒 P99。
- **序列化和日志**：响应 JSON codec 累计约 6.03% CPU，Zap Info 约 2.35%。mutex 累计等待约 5.68 秒，`Rows.close` 路径约占 45.61%、取连接路径约 20.68%、Zap writer 约 17.81%。这是累计争用等待，不能与 CPU 秒相加，也不能据此声称日志是主瓶颈。
- **block 不等于业务阻塞**：block 累计约 179.73 秒，其中 MySQL `startWatcher` 占约 **85.86%**。它等待连接完成或取消，包含正常生命周期等待；这个比例不能解释为“85.86% 请求被数据库锁住”。goroutine 合并数据中约 66.25% 是 HTTP Reader 等下一次请求的栈，同样不能当作业务卡死。

**可以确认的瓶颈线索**是三次串行数据库访问的成本、结果扫描、重复构造查询对象产生的分配与 GC，以及尾段连接等待和共享 CPU 调度竞争。**尚不能确认的根因**是 MySQL 服务端具体执行步骤、数据库锁、单线程限制，或某项硬件的持续硬上限；本轮没有 MySQL 内部执行/锁等待数据。

## Grafana 不同 CPU 面板的 100% 含义

实际线上查询和单位决定口径，不能仅按标题判断。

- **HostMonitor / CPU Busy**：`100 * (1 - avg(rate(node_cpu_seconds_total{mode="idle"}[1m])))`。对所有逻辑核取平均，**100% 对应全部 4 核的可计时容量都处于非 idle 状态**。1 核满载、其余 3 核完全闲置约为 25%。非 idle 还包含 iowait 等时间，因此 100% 也不能自动等同于四核都在执行应用计算。
- **HostMonitor / CPU Basic、CPU**：各 mode 时间先求和再除以核数，单位为 `percentunit`，原值 1 显示 100%。同样按全机归一化；堆叠图包含 Idle 时，各模式之和接近 100% 是时间构成完整，不是 CPU 满载。
- **HostMonitor / CPU Saturation per Core**：按 `cpu` 标签逐核显示 Running、Waiting Queue 及等待占比。等待时间可由多个任务累计，Waiting Queue 可以超过 100%，不是硬件利用率超过上限。
- **HostMonitor / Exporter Process CPU Usage**：`irate(process_cpu_seconds_total[1m])`，`percentunit`，对应 **node-exporter 进程**。100% 表示约 1 核 CPU 时间，不是整个主机满载。
- **PGO Application / Process CPU**：`irate(process_cpu_seconds_total{job="pgo-app"}[1m])`，单位是 `short`，实际显示的是**核心数**，不是百分比。**1 = 1 核满载等效 CPU 时间；4 = 4 核容量**。本轮 0.89 约等于单核口径 89%，或全机口径 22.3%；并不意味着固定某一个物理核跑到 89%。
- **Docker Monitor / CPU Usage**：`sum(rate(container_cpu_usage_seconds_total[5m])) by (name) * 100`，没有除以主机核数。**100% = 1 核，400% = 4 核等效容量**。线上面板还有 5 分钟平滑，会混入本次正式负载之外的数据；本报告定位一分钟使用补充 `irate[1m]`，不能把它的峰值当成线上 5 分钟曲线值。

node-exporter 的 CPU 时间按秒计数，rate 后得到每秒 CPU 时间，见 [Prometheus 官方说明](https://prometheus.io/docs/guides/node-exporter/)。Pyroscope 数据通过[官方公开查询 API](https://grafana.com/docs/pyroscope/latest/reference-server-api/)取得，没有读取其存储内部格式。

## 核验

六类 pprof JSON 的样本总量与同窗口 Pyroscope flamegraph total 逐项一致；所有保存的 JSON 可解析。压测数字按当前三个报告核对，时间换算与源采样时间核对。未修改服务配置或业务实现；所有本轮请求进程已退出。

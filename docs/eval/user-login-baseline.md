# 多用户注册与登录基线

> 专题中枢：[登录性能测试闭环](../design/2026-09-30-01-login-performance-hub.md)
>
> 本文记录可重复的实验条件和结果。批次清单、token、Vegeta 二进制结果与 profile 只保存在 `docs/eval/performance/`，不提交仓库。

## 1. 前置条件

- 使用带 MySQL 的 userService 配置，HTTP 和 diagnostics 端口仅暴露在受控环境。
- 将 `Diagnostics.Enabled` 和 `Diagnostics.Pprof` 设为 `true`；配置 block、mutex 采样参数。CPU、heap 默认由 Alloy 持续写入 Pyroscope。
- 使用 `make build` 构建仓库。
- 安装固定版本 Vegeta：

```shell
go install github.com/tsenart/vegeta/v12@v12.13.0
```

`pgo performance login` 会检查 Vegeta 二进制中的模块版本，不符合 `v12.13.0` 时会在创建测试用户前停止。

默认 profile 来源还需要安装官方 `profilecli v2.2.0`。程序会在创建测试用户前检查版本。没有运行 Alloy/Pyroscope 时，可通过 `--profile-source pprof` 改为直接从应用诊断端口采集。

服务由维护者使用本地配置在前台启动，结束时按 Ctrl+C，不使用 `nohup` 或无人管理的后台进程。

## 2. 一条命令执行完整闭环

`pgo` 默认无参数启动时进入交互菜单，但 Cobra 子命令支持一次传入全部参数，不需要回答交互问题：

```shell
./bin/pgo performance login \
  --api http://127.0.0.1:20000 \
  --pprof http://127.0.0.1:20002/debug/pprof/ \
  --pyroscope http://127.0.0.1:24040 \
  --grafana http://127.0.0.1:23000 \
  --profile-source pyroscope \
  --profile-service pgo-app \
  --users 100 \
  --concurrency 10 \
  --rps 10 \
  --warmup 10s \
  --duration 60s \
  --timeout 5s \
  --output docs/eval/performance/login-100u-10rps \
  > docs/eval/performance/login-100u-10rps-cli-output.txt
```

最后一行的 `>` 是 shell 重定向，只把本次 CLI 的阶段进度和产物说明保存到文件，不属于 `pgo` 的功能参数。错误仍输出到终端，便于立即发现失败。

命令内部按以下顺序执行：

1. **准备用户**：通过真实 `POST /user/token` 创建本轮专用用户，并在每次成功后原子更新批次清单。
2. **验证正确性**：逐个重新登录，核对身份与 token，验证受保护接口拒绝无效鉴权。该步骤防止把数据错误误判为性能问题。
3. **生成目标**：把批次用户转换成 Vegeta JSON targets。这一步只准备请求定义，不产生正式负载。
4. **预热**：按目标 RPS 运行 Vegeta，预热结果不计入正式报告。
5. **正式负载与观测**：Vegeta 产生登录负载；程序保存负载前后 metrics，自动开启限时诊断并采集 goroutine、block、mutex。默认使用 `profilecli` 导出同一时间窗的 CPU、heap；`--profile-source pprof` 改为 HTTP 直采。只有显式传入 `--runtime-trace` 才会额外采集最长 10 秒的 runtime trace。
6. **生成报告**：输出 Vegeta 原始结果、文本报告和带指标含义的综合摘要。
7. **清理用户**：只删除批次清单中的用户。前面任一步骤失败时也会尝试清理，并保留已经生成的诊断文件。

先完成 100 用户、10 RPS 的小档闭环，再分别调整 `--users` 和 `--rps` 执行 1,000、10,000 用户以及 50、100 RPS。一次只运行一个批次，每轮使用不同输出目录。

## 3. 输出文件

命令结束时会先打印一行输出目录，再逐项打印文件名及用途，不在每个文件名前重复父目录。每个输出目录包含：

- `00-run.json`：本轮输入参数，用于复现。
- `01-users.json`：批次用户清单，用于验证与精确清理，包含 token，不得提交。
- `02-login-targets.jsonl`：Vegeta 登录请求定义，每行一条 JSON 请求。
- `10-vegeta-results.bin`：Vegeta 原始请求结果。
- `11-vegeta-report.txt`：吞吐、成功率和延迟分位数报告。
- `20-metrics-after.prom`：正式负载结束后的 Prometheus 指标。
- `21-metrics-before.prom`：正式负载开始前的 Prometheus 指标。
- `30-cpu.pprof`：正式负载期间持续采集的 CPU profile。
- `31-goroutine.pprof`：正式负载期间的 goroutine 快照。
- `32-heap.pprof`：正式负载期间的堆快照。
- `33-block.pprof`：正式负载期间的阻塞 profile。
- `34-mutex.pprof`：正式负载期间的锁竞争 profile。
- `35-runtime.trace`：仅在 `--runtime-trace` 开启时生成的运行时 trace。
- `40-summary.md`：格式化的 Vegeta 结果、重点服务指标及其意义。

如果使用示例中的 shell 重定向，输出目录旁还会有 `login-100u-10rps-cli-output.txt`。它记录自动化程序自身的阶段进度与产物说明，由执行命令的 shell 创建。

## 4. 指标阅读边界

`40-summary.md` 聚焦本轮登录压测直接需要的证据：

- Vegeta throughput、success、P50、P95、P99 说明客户端看到的吞吐、成功率和延迟尾部。
- `pgo_http_*` 说明服务端 HTTP 请求量、错误和耗时。
- `pgo_db_query_*` 说明数据库查询量和耗时。
- `pgo_db_connections_*` 说明连接池的打开、使用中、空闲和等待状态。
- `go_*` 说明 Go 运行时内存、GC 和 goroutine 状态。
- `process_*` 说明服务进程的 CPU 与常驻内存。

负载前后 metrics 是同一轮正式负载的边界快照。计数器增长反映区间内累计工作量，gauge 反映采样时刻状态；CPU、heap、goroutine、block 和 mutex profile 为后续规则化判断保留输入，本阶段仍以实际基线为准，不预设问题阈值。

## 5. 实验记录

### 环境

- 日期：待填写
- Git 提交：待填写
- Go 版本：待填写
- MySQL 版本：待填写
- Vegeta 版本：v12.13.0
- profilecli 版本：v2.2.0
- 配置摘要：待填写，仅记录超时、连接池和 diagnostics 开关

### 条件

- 用户数：待填写
- 注册并发：待填写
- 登录 RPS：待填写
- 预热、持续时间和超时：待填写
- 输出目录：待填写

### 结果

- 注册与验证结果：待填写
- 登录吞吐、成功率、P50/P95/P99：待填写
- CPU、内存和 goroutine：待填写
- 查询耗时与连接池：待填写
- profile 摘要：待填写

### 阶段性结论

- 现象：待填写
- 假设：待填写
- 证据：待填写
- 定位：待填写
- 后续动作：有明确证据后登记新的修复任务；没有证据时不修改业务实现。

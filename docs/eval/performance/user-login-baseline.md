# 多用户注册与登录基线

> 本文记录可重复的实验条件和结果。批次清单、token、Vegeta 二进制结果与 profile 只保存在 `.local/performance/`，不提交仓库。

## 1. 前置条件

- 使用带 MySQL 的本地 userService 配置，HTTP 和 diagnostics 端口仅暴露在受控环境。
- 将 `Diagnostics.Enabled` 和 `Diagnostics.Pprof` 设为 `true`，完成实验后按需关闭 pprof。
- 构建仓库：`GOTOOLCHAIN=local make build`。
- 安装固定版本 Vegeta：

```shell
GOTOOLCHAIN=local go install github.com/tsenart/vegeta/v12@v12.13.0
vegeta -version
```

服务由维护者使用本地配置在前台启动，结束时按 Ctrl+C，不使用 `nohup` 或无人管理的后台进程。

## 2. 准备与验证用户

先从 100 用户小档开始。工具通过真实 HTTP 注册用户，每个成功响应都会原子更新本地清单，进程中断后仍可按清单清理。

```shell
./bin/pgo user-load prepare \
  --base-url http://127.0.0.1:8000 \
  --batch small01 \
  --count 100 \
  --concurrency 10 \
  --output .local/performance/small01-users.json

./bin/pgo user-load verify \
  --manifest .local/performance/small01-users.json \
  --concurrency 10

./bin/pgo user-load targets \
  --manifest .local/performance/small01-users.json \
  --output .local/performance/small01-login-targets.json
```

`verify` 会重新登录每个用户，核对返回身份，并用 token 调用受保护的用户查询接口；它也会确认无 token 和无效 token 的请求被拒绝。

中档和大档分别把 `--count` 改为 `1000`、`10000`，并使用新的批次 ID 和文件名。一次只运行一个批次，确认小档完整闭环后再扩大。

## 3. 登录负载

每档正式负载前先预热 10 秒：

```shell
vegeta attack \
  -format=json \
  -targets=.local/performance/small01-login-targets.json \
  -rate=10/s \
  -duration=10s \
  -timeout=5s > /dev/null
```

依次执行 10、50、100 RPS，每轮持续 60 秒。以下以 10 RPS 为例，其他档位只替换 `rate` 和输出文件名：

```shell
vegeta attack \
  -format=json \
  -targets=.local/performance/small01-login-targets.json \
  -rate=10/s \
  -duration=60s \
  -timeout=5s \
  > .local/performance/small01-10rps.bin

vegeta report \
  -type=text \
  .local/performance/small01-10rps.bin \
  > .local/performance/small01-10rps.txt
```

负载命令在前台运行。需要同时采集 profile 时，在另一个终端执行诊断命令。

## 4. 指标与 profile

诊断地址默认是 `http://127.0.0.1:19090`：

```shell
./bin/pgo diagnostics metrics-url

./bin/pgo diagnostics profile cpu \
  --seconds 30 \
  --output .local/performance/small01-10rps-cpu.pprof

./bin/pgo diagnostics profile heap \
  --output .local/performance/small01-10rps-heap.pprof

./bin/pgo diagnostics profile goroutine \
  --output .local/performance/small01-10rps-goroutine.pprof
```

每轮至少记录：

- `pgo_http_requests_total` 和 `pgo_http_request_duration_seconds`。
- `go_*`、`process_*` 运行时与进程指标。
- `pgo_db_query_duration_seconds`。
- `pgo_db_connections_open`、`pgo_db_connections_in_use`、`pgo_db_connections_idle`。
- `pgo_db_connection_wait_total`、`pgo_db_connection_wait_duration_seconds_total`。
- Vegeta 实际吞吐、成功率、P50、P95、P99 和错误摘要。

## 5. 清理

每轮完成或中断后，只按本批次清单删除用户。清理可重复执行：

```shell
./bin/pgo user-load cleanup \
  --manifest .local/performance/small01-users.json \
  --concurrency 10
```

## 6. 实验记录

### 环境

- 日期：待填写
- Git 提交：待填写
- Go 版本：待填写
- MySQL 版本：待填写
- Vegeta 版本：v12.13.0
- 配置摘要：待填写，仅记录超时、连接池和 diagnostics 开关

### 条件

- 批次 ID：待填写
- 用户数：待填写
- 注册并发：待填写
- 登录 RPS：待填写
- 预热、持续时间和超时：10 秒、60 秒、5 秒
- 采样起止时间：待填写

### 结果

- 注册成功数、失败数和耗时：待填写
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

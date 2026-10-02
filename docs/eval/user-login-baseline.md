# 多用户注册与登录基线

> 专题中枢：[登录性能测试闭环](../design/2026-09-30-01-login-performance-hub.md)
>
> 本文记录可重复的负载条件和客户端结果。批次清单、token 与 Vegeta 结果只保存在 `.local/performance/login/`，不提交仓库。

## 1. 前置条件

- 使用带 MySQL 的 userService 配置，并确保导航页和 API 可访问。
- 使用 `make build` 构建仓库。
- 安装固定版本 Vegeta：

```shell
go install github.com/tsenart/vegeta/v12@v12.13.0
```

`pgo performance login` 会检查 Vegeta 版本，不符合 `v12.13.0` 时在创建测试用户前停止。Grafana 与 Pyroscope 独立承担服务指标、日志和 profiling，不是压测命令的运行依赖。

## 2. 单档模式

数字形式的 `--rps` 只运行一个压力等级：

```shell
./bin/pgo performance login http://127.0.0.1:20080 --rps 50
```

命令依次完成测试用户准备与验证、Vegeta targets 生成、预热、正式负载、负载报告和用户清理。前面任一步骤失败时仍尝试精确清理已创建用户。

## 3. 自动升压模式

省略 `--rps` 后依次运行内置的 `10、25、50、100、200、500 RPS`：

```shell
./bin/pgo performance login http://127.0.0.1:20080
```

每次运行在 `.local/performance/login/` 下创建独立目录，各档写入 `rps-010/` 至 `rps-500/`。某档执行失败或出现非成功响应时停止后续升压，保留已有负载产物并完成用户清理。

自动模式根目录包含：

- `00-auto-run.json`：固定阶梯和运行模式。
- `40-auto-results.json`：逐档请求数、吞吐、成功率、P50、P95、P99 与错误。
- `41-auto-summary.md`：同一组负载指标的可读跨档汇总。

## 4. 单档输出文件

- `00-run.json`：本轮负载参数。
- `01-users.json`：测试用户清单，包含 token，不得提交。
- `02-login-targets.jsonl`：Vegeta 登录请求定义。
- `10-vegeta-results.bin`：Vegeta 原始请求结果。
- `11-vegeta-report.txt`：请求数、吞吐、成功率和延迟分位数。

服务端 HTTP、数据库、Go runtime、CPU、内存、日志和 profile 直接在 Grafana 与 Pyroscope 中按压测时间范围查看，不下载到本地结果目录。

## 5. 指标阅读边界

performance 只记录负载发起方直接测得的结果：

- requests 表示本轮实际完成的请求数量。
- throughput 表示实际完成速率。
- success 表示成功响应比例。
- P50、P95、P99 表示客户端观察到的延迟分布。

这些结果描述负载本身，不代替服务端指标或 profile。需要定位服务内部瓶颈时，使用 Grafana 和 Pyroscope 查看同一时间范围。

## 6. 实验记录

### 环境

- 日期：待填写
- Git 提交：待填写
- Go 版本：待填写
- MySQL 版本：待填写
- Vegeta 版本：v12.13.0
- 配置摘要：待填写，仅记录会影响负载结果的服务配置

### 条件

- 用户数：待填写
- 登录 RPS：待填写
- 预热、持续时间和请求超时：待填写
- 输出目录：待填写
- Grafana/Pyroscope 时间范围：待填写

### 结果

- 注册与验证结果：待填写
- 请求数、吞吐、成功率、P50/P95/P99：待填写
- Grafana/Pyroscope 观察：按需记录结论，不复制平台原始数据

### 阶段性结论

- 现象：待填写
- 结论：待填写

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
./bin/pgo performance login http://127.0.0.1:20080 \
  --output-dir .local/performance/login/comparison --rps 50 --duration 60s
```

命令依次完成测试用户准备与验证、Vegeta targets 生成、预热、正式负载和负载报告，默认保留用户。指定同一 `--output-dir` 会自动复用完整的 100 人批次；规模不符、准备或清理未完成、实际用户缺失时，先精确清理旧批次再重建。HTTP 检查失败时停止，不自动删除。菜单中的登录性能测试和登录数据清理共享输出目录缓存。
`--duration` 使用 Go 时长格式并控制每档正式负载，默认为 `60s`；预热时长由性能框架固定管理。

## 3. 自动升压模式

省略 `--rps` 后依次运行内置的 `200、400、600、800、1000 RPS`，每档共用同一 `--duration`：

```shell
./bin/pgo performance login http://127.0.0.1:20080
```

CLI 省略 `--output-dir` 时在 `.local/performance/login/` 下创建独立目录，整轮只准备一次用户和 targets，各档报告写入 `rps-200/` 至 `rps-1000/`。某档执行失败或出现非成功响应时停止后续升压，保留已有负载产物与测试用户。

自动模式根目录保存 `00-auto-run.json` 固定阶梯输入、用户清单及 targets。各档只保存自身负载报告，不生成跨档汇总，效果对比在 Grafana 与 Pyroscope 中观察。

## 4. 单档输出文件

- `00-run.json`：本轮负载参数。
- `01-users.json`：测试用户清单，包含 token，不得提交。
- `02-login-targets.jsonl`：Vegeta 登录请求定义。
- `10-vegeta-results.bin`：Vegeta 原始请求结果。
- `11-vegeta-report.txt`：请求数、吞吐、成功率和延迟分位数。

服务端 HTTP、数据库、Go runtime、CPU、内存、日志和 profile 直接在 Grafana 与 Pyroscope 中按压测时间范围查看，不下载到本地结果目录。

## 5. 独立清理

```shell
./bin/pgo performance login-cleanup .local/performance/login/comparison
# 同样可用场景子命令
./bin/pgo performance login cleanup .local/performance/login/comparison
```

清理仅需要测试输出目录，从清单获取服务地址并重新登录刷新令牌；按批次名称恢复未写入清单的用户，逐条删除并保存进度。失败后可重复执行，清理完成后再执行不创建用户。清理不要求 Vegeta 或 portal 可用，批次 HTTP 服务仍需可访问。

## 6. 指标阅读边界

performance 只记录负载发起方直接测得的结果：

- requests 表示本轮实际完成的请求数量。
- throughput 表示实际完成速率。
- success 表示成功响应比例。
- P50、P95、P99 表示客户端观察到的延迟分布。

这些结果描述负载本身，不代替服务端指标或 profile。需要定位服务内部瓶颈时，使用 Grafana 和 Pyroscope 查看同一时间范围。

## 7. 实验记录

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

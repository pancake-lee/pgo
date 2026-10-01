# 登录性能测试闭环专题中枢

> 本文档是登录性能测试闭环的唯一中枢。初始闭环见 [backlog 任务 23](../backlog.md#23-多用户注册与登录的-http-压测及观测闭环)，持续 profiling 与双来源采集演进见 [任务 28](../backlog.md#28-alloypyroscope-与受控运行时诊断)，逐级加压与代码分层见 [任务 29](../backlog.md#29-登录场景逐级加压与监控验证自动化)。

## 关联产物

- [任务 23](../backlog.md#23-多用户注册与登录的-http-压测及观测闭环)：需求、方案、任务列表与验收。
- [任务 28](../backlog.md#28-alloypyroscope-与受控运行时诊断)：Alloy、Pyroscope、受控运行时诊断及性能工具双来源采集。
- [任务 29](../backlog.md#29-登录场景逐级加压与监控验证自动化)：性能测试分层、单档兼容与内置阶梯自动升压。
- [多用户注册与登录基线](../eval/user-login-baseline.md)：当前可执行的手工实验步骤与记录模板。
- `cmd/pgo/performance/login/`：登录场景的用户批次准备、验证、Vegeta targets 生成与清理。
- `pkg/papp/observability.go`：长期 CPU/heap 与限时 goroutine、block、mutex、trace 诊断端点。

## 时间线

- 第一阶段：建立 `user-load prepare/verify/targets/cleanup`、diagnostics、数据库指标和手工 Vegeta 基线，离线测试与构建已通过，等待真实 MySQL 环境验收。
- 2026-09-29：基线文档补充七步闭环说明，明确每条命令的输入、输出和作用。
- 2026-09-30：规划第二阶段自动化，新增 `pgo performance login`，由 Go 统一编排外部 Vegeta CLI、metrics、pprof、报告和清理；取消独立 `user-load` tools 子项，其能力改为登录场景内部阶段。
- 2026-10-01：任务 28 将 CPU、heap 改为 Alloy 持续写入 Pyroscope；性能工具默认使用 `profilecli` 导出负载时间窗，也保留 HTTP 直采模式，并自动采集限时 goroutine、block、mutex。
- 2026-10-01：任务 29 规划整理性能测试分层；保留数字 `--rps` 单档模式，并以 `--rps auto` 运行内置 `10、25、50、100、200、500 RPS` 阶梯和跨档汇总。

## 当前完成度

- 用户批次生命周期：已实现并完成自动验证。
- 服务端 metrics 与 pprof 下载：已实现并完成自动验证。
- 手工闭环文档：已完成。
- `performance login` 全自动编排：已完成并通过自动验证。
- 真实 MySQL 环境闭环：已完成 100 用户、10 RPS、60 秒代表性验证。
- 持续 profiling 与两种 profile 来源：代码和自动验证已完成，等待宿主机 Compose 真实链路验收。
- 单档与自动阶梯双模式：代码和自动验证已完成，等待全新部署环境运行确认。

## 下一轮建议

在全新部署环境运行任务 29 的自动阶梯并形成首份跨档基线。新的压测对象仍作为 `performance` 下的独立子命令逐项规划，只复用已经由登录场景验证的通用执行层。

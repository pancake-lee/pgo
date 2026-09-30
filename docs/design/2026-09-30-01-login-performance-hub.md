# 登录性能测试闭环专题中枢

> 本文档是登录性能测试闭环的唯一中枢。具体方案与任务拆分以 [backlog 任务 23](../backlog.md#23-多用户注册与登录的-http-压测及观测闭环) 为准。

## 关联产物

- [任务 23](../backlog.md#23-多用户注册与登录的-http-压测及观测闭环)：需求、方案、任务列表与验收。
- [多用户注册与登录基线](../eval/user-login-baseline.md)：当前可执行的手工实验步骤与记录模板。
- `cmd/pgo/tools/performance/login/`：登录场景的用户批次准备、验证、Vegeta targets 生成与清理。
- `cmd/pgo/tools/diagnostics/`：metrics 地址与 CPU、heap、goroutine profile 下载。

## 时间线

- 第一阶段：建立 `user-load prepare/verify/targets/cleanup`、diagnostics、数据库指标和手工 Vegeta 基线，离线测试与构建已通过，等待真实 MySQL 环境验收。
- 2026-09-29：基线文档补充七步闭环说明，明确每条命令的输入、输出和作用。
- 2026-09-30：规划第二阶段自动化，新增 `pgo performance login`，由 Go 统一编排外部 Vegeta CLI、metrics、pprof、报告和清理；取消独立 `user-load` tools 子项，其能力改为登录场景内部阶段。

## 当前完成度

- 用户批次生命周期：已实现并完成自动验证。
- 服务端 metrics 与 pprof 下载：已实现并完成自动验证。
- 手工闭环文档：已完成。
- `performance login` 全自动编排：已完成并通过自动验证。
- 真实 MySQL 环境闭环：已完成 100 用户、10 RPS、60 秒代表性验证。

## 下一轮建议

按需使用现有命令补充 1,000/10,000 用户和 50/100 RPS 实验记录。新的压测对象作为 `performance` 下的独立子命令逐项规划，不提前抽象业务场景公共模型。

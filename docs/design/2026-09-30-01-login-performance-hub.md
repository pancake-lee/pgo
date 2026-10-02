# 登录性能测试闭环专题中枢

> 本文档是登录性能测试闭环的唯一中枢。登录负载能力见 [任务 29](../backlog.md#29-登录场景逐级加压与负载结果记录)，独立观测平台见 [任务 28](../backlog.md#28-alloypyroscope-与受控运行时诊断)。

## 关联产物

- [任务 23](../backlog.md#23-多用户注册与登录的-http-负载闭环)：登录负载场景的初始实现。
- [任务 28](../backlog.md#28-alloypyroscope-与受控运行时诊断)：Grafana、Prometheus、Loki、Alloy、Pyroscope 与受控诊断。
- [任务 29](../backlog.md#29-登录场景逐级加压与负载结果记录)：登录单档负载、固定阶梯与负载结果。
- [多用户注册与登录基线](../eval/user-login-baseline.md)：当前实验方法与结果记录模板。
- `cmd/pgo/performance/login/`：测试用户准备、验证、targets 生成与清理。
- `cmd/pgo/common/`：导航页服务清单解析与固定组件 URL。
- `cmd/pgo/performance/`：Vegeta 负载执行与负载侧结果记录。

## 职责边界

- performance 负责准备场景、制造负载、保存 Vegeta 结果和清理测试数据。
- 单档与跨档结果只包含请求数、实际吞吐、成功率、P50、P95、P99 和执行错误。
- Grafana 与 Pyroscope 负责服务指标、日志和 profiling；performance 不抓取、导出或复制平台数据。
- 两侧通过压测时间范围关联，不建立本地观测产物的第二套生命周期。

## 当前完成度

- 用户批次准备、验证、targets 与精确清理：已实现并完成自动验证。
- 单档与固定六档自动升压：已实现并完成自动验证。
- Vegeta 负载结果与跨档汇总：已收口为纯负载侧字段。
- 导航服务发现：已迁移到 CLI 公共包并补齐全部固定组件 URL。
- Grafana、Prometheus、Loki、Alloy 与 Pyroscope：代码和部署配置已完成自动验证，等待宿主机真实链路验收。
- 登录场景 API：批次、用户数、超时和并发策略已收回包内。

## 下一轮建议

在真实环境运行固定阶梯负载；客户端结果读取 Vegeta 产物，服务状态、指标和 profile 直接在 Grafana 与 Pyroscope 中按压测时间范围查看。

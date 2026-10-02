# 登录性能测试闭环专题中枢

> 本文档是登录性能测试闭环的唯一中枢。性能框架与登录场景解耦见 [任务 30](../backlog.md#30-性能框架与登录场景解耦)，独立观测平台见 [任务 28](../backlog.md#28-alloypyroscope-与受控运行时诊断)。

## 关联产物

- [任务 23](../backlog.md#23-多用户注册与登录的-http-负载闭环)：登录负载场景的初始实现。
- [任务 28](../backlog.md#28-alloypyroscope-与受控运行时诊断)：Grafana、Prometheus、Loki、Alloy、Pyroscope 与受控诊断。
- [任务 29](../backlog.md#29-登录场景逐级加压与负载结果记录)：登录单档负载、固定阶梯与负载结果。
- [任务 30](../backlog.md#30-性能框架与登录场景解耦)：通用单档负载、自动升压与登录场景注册边界。
- [任务 32](../backlog.md#32-性能场景公共参数收口到-core)：由 core 统一读取和解析 portal URL、RPS 与 duration。
- [多用户注册与登录基线](../eval/user-login-baseline.md)：当前实验方法与结果记录模板。
- `cmd/pgo/performance/login/`：登录命令、交互入口、测试用户准备、验证、targets 生成、清理与登录自动化策略。
- `cmd/pgo/common/`：导航页服务清单解析与固定组件 URL。
- `cmd/pgo/performance/`：Performance 分组入口、Vegeta 单档负载、通用自动升压与负载侧结果记录。

## 职责边界

- performance 根包负责登记分组工具，core 负责制造单档负载并记录 Vegeta 结果；场景包负责准备和清理测试数据。
- core 负责所有场景共用的 portal URL、RPS 和 duration 输入与校验；场景不自行定义这三项参数。
- 自动升压只组织压力档位，每个档位复用 performance 的单档执行链。
- 单档与跨档结果只包含请求数、实际吞吐、成功率、P50、P95、P99 和执行错误。
- Grafana 与 Pyroscope 负责服务指标、日志和 profiling；performance 不抓取、导出或复制平台数据。
- 两侧通过压测时间范围关联，不建立本地观测产物的第二套生命周期。

## 当前完成度

- 用户批次准备、验证、targets 与精确清理：已实现并完成自动验证。
- 单档与固定六档自动升压：已实现并完成自动验证。
- Vegeta 负载结果与跨档汇总：已收口为纯负载侧字段。
- 导航服务发现：已迁移到 CLI 公共包并补齐全部固定组件 URL，无协议的主机与端口默认补全 `http://`。
- Grafana、Prometheus、Loki、Alloy 与 Pyroscope：代码和部署配置已完成自动验证，等待宿主机真实链路验收。
- 登录场景 API：批次、用户数、超时和并发策略已收回包内。
- 性能框架与登录场景解耦：已完成 pclient 两层菜单登记、通用单档执行和自动升压回调，并完成自动验证。
- 性能代码可读性：已统一行宽、函数声明分级换行、函数调用整行或逐实参换行、赋值与判断分行及执行阶段留白，对应规则已沉淀到 Harness。
- 两层菜单：performance 已删除专属场景注册和双运行模式菜单，由 `pkg/pclient` 的通用两层分组承载；pgo 定义 Performance 一级分组，`performance/menu.go` 登记 login 二级工具，通用负载框架位于 `performance/core`，login 只处理自身命令参数与执行。
- 公共参数边界：portal URL、RPS 与 duration 的 Cobra/交互输入、解析及执行编排已收口到 core，login 已收缩为场景定义与准备清理。

## 下一轮建议

后续新增场景时，验证只提供场景定义和准备器即可接入。真实环境运行时，客户端结果读取 Vegeta 产物，服务状态、指标和 profile 直接在 Grafana 与 Pyroscope 中按压测时间范围查看。

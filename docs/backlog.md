# Backlog

> 全部技术需求池，按优先级排列。
> 状态标记：Done、Pending、WIP、Approved、Abandoned、Rejected

| 状态 | 序号 | 类别 | 任务 | 简述 |
|------|------|------|------|------|
| Done | 1 | 核心框架 | 组织架构树 | 一个用户可加入多个部门，在不同部门分别担任不同职位 |
| Done | 2 | 核心框架 | genCURD 代码生成工具 | 通过 ORM 结构体生成基础 CURD 代码，从接口到数据库完整链路 |
| Done | 3 | 客户端 | 换课功能 | 教师换课/欠课管理，含课程表解析、候选课堂计算、换课记录 CURD |
| Done | 4 | 核心框架 | 任务管理后端 | task 表 CRUD 接口自动生成，支持树形结构和表格视图 |
| Pending | 5 | 集成 | 多维表格双向同步冲突规则 | 定义双端同时修改的优先级、版本号/时间戳策略、回环标记规范 |
| Pending | 6 | 集成 | 多维表格同步可观测性 | 重试队列、失败补偿、告警与审计日志 |
| Pending | 7 | 集成 | 多维表格多平台适配层 | 统一 APITable/飞书/企微等多维表格接入接口 |
| Pending | 8 | 集成 | TEMP 标记方向区分 | 修复多维表格双向同步中 TEMP 标记不区分方向导致的回环误判 BUG |
| Pending | 9 | 基础设施 | 排查 BUG 案例建设 | 编写含延时/日志/随机错误/并发的测试接口，RequestID 串联日志 |
| Pending | 10 | 基础设施 | 日志策略优化 | 探究生产环境日志打印策略，内存缓存日志不出错不打印 |
| Pending | 11 | 基础设施 | 链路追踪 | 引入 Jaeger，确认 gRPC 接口调用链条 |
| Pending | 12 | 基础设施 | pprof 性能分析 | 提供 pprof 信息，确认接口内部函数调用链条 |
| Pending | 13 | 基础设施 | 持续监控 | 接口调用频率/耗时/成功率、硬件数据、服务健康状态，用 pgo CLI 承载 |
| Pending | 14 | 代码生成 | genCURD inferServiceName 配置化 | 支持通过配置文件或命令行参数（如 `-svc-map "photos:photo,import_jobs:import"`）映射表名到 service 名，替代当前硬编码返回 "default" 的行为 |
| Pending | 15 | 应用框架 | papp.AppCtx 中间件能力增强 | 增加 TraceID（从 HTTP header 或自动生成）、StartTime（请求耗时日志）、补充 context.Context 嵌合方法 |
| Pending | 16 | 代码生成 | genCURD 支持多主键表 | 当前多主键时跳过 Update/Delete 生成，后续支持联合主键的 WHERE 条件包含所有主键字段 |
| Pending | 17 | 代码生成 | genGORM 与 genCURD 合并 | 增加 `pgo genAll` 一键完成 GORM + CURD 生成，减少数据库连接开销。短期分开生成有利于排查问题 |

---

## 详细说明

### 5. 多维表格双向同步冲突规则

定义并实现当本地和远端同时修改同一数据时的处理策略，包括优先级规则、版本号或时间戳比对策略、回环标记规范，确保同步不会丢失或覆盖数据。

### 6. 多维表格同步可观测性

为同步流程建立可观测性体系：失败重试队列、数据补偿机制、告警规则以及完整的审计日志，支撑后续跨业务推广。

### 7. 多维表格多平台适配层

抽象统一的适配层接口，让 APITable、飞书多维表格、企微智能表格等多种多维表格产品可以通过同一套接入逻辑对接。

### 8. TEMP 标记方向区分

当前多维表格双向同步中 TEMP 标记不区分 local→remote 和 remote→local 两个方向，导致特定场景下回环误判，中断正常的同步逻辑。需要拆分方向标记并实现幂等与回环抑制。

### 9. 排查 BUG 案例建设

编写一个测试接口，包含 3 个内部接口调用，内置延时/日志/随机错误/并发，让日志产生"交错"。报错时在 response 中返回 RequestID，通过该 key 在日志平台快速定位所有相关日志。进一步探究生产环境的日志最佳实践。

### 10. 日志策略优化

探究生产环境中接口应打印哪些数据，平衡性能和排查需求。考虑内存缓存日志、不出错不打印、仅 warn/error 级别输出的策略。

### 11. 链路追踪

引入 Jaeger 等链路追踪工具，确认 gRPC 接口调用链条，可视化服务间依赖。

### 12. pprof 性能分析

提供 pprof 端点，支持分析接口内部的函数调用链条和性能热点。

### 13. 持续监控

除了事后排查的日志平台，建立持续监控体系：接口调用频率、耗时、成功率、硬件数据、中间件和服务进程健康状态、自动业务状态。控的方面倾向用 pgo CLI 承载交互操作。

### 14. genCURD inferServiceName 配置化

当前 `inferServiceName()` 函数硬编码返回 `"default"`，所有表归入同一个 service。对于多表分服务的项目不适用。需要支持通过配置文件或命令行参数（如 `-svc-map "photos:photo,import_jobs:import"`）读取表名到 service 名的映射关系。

> 来源：photo-agent backend 重构方案 5.2，暂缓。

### 15. papp.AppCtx 中间件能力增强

当前 `AppCtx` 包含 `UserId`、`Log`、缓存 map，但没有 request-scoped 的 trace ID、请求计时等。需要增加：
- `TraceID` 字段（从 HTTP header 或自动生成）
- `StartTime` 字段，方便在日志中输出请求耗时
- 确认 `context.Context` 嵌合是否有遗漏的方法

> 来源：photo-agent backend 重构方案 5.5，暂缓，等项目实际需要再扩展。

### 16. genCURD 支持多主键表

当前检测到多主键时直接 `tbl.PriCol = nil` 跳过，生成代码不含 Update/Delete。需要支持联合主键的 Update/Delete 生成（WHERE 条件包含所有主键字段）。

> 来源：photo-agent backend 重构方案 5.7，P3 后续优化。

### 17. genGORM 与 genCURD 合并

分两步执行 `make gorm` 再 `make curd`，中间需要保持数据库连接。两个命令各自连接一次数据库。建议增加 `pgo genAll` 命令一次性完成。但短期来看分开生成有利于排查问题，暂不合并。

> 来源：photo-agent backend 重构方案 5.8，P3 后续优化。

# Backlog

> 全部技术需求池，按优先级排列。
> 状态标记：Done、Pending、WIP、Approved、暂缓、Abandoned、Rejected

| 状态 | 序号 | 类别 | 任务 | 简述 |
|------|------|------|------|------|
| Done | 1 | 核心框架 | 组织架构树 | 一个用户可加入多个部门，在不同部门分别担任不同职位 |
| Done | 2 | 核心框架 | genCURD 代码生成工具 | 通过 ORM 结构体生成基础 CURD 代码，从接口到数据库完整链路 |
| Done | 3 | 客户端 | 换课功能 | 教师换课/欠课管理，含课程表解析、候选课堂计算、换课记录 CURD |
| Done | 4 | 核心框架 | 任务管理后端 | task 表 CRUD 接口自动生成，支持树形结构和表格视图 |
| Approved | 12 | 基础设施 | pprof 性能分析 | 已确认未注册 pprof 端点；仅有间接依赖，不能用于分析 |
| Approved | 13 | 基础设施 | 持续监控 | 基础监控栈已存在，但没有应用指标、健康检查和 CLI 查询入口 |
| Approved | 14 | 代码生成 | genCURD inferServiceName 配置化 | 已确认规则硬编码在生成器中，未提供外部映射配置 |
| Approved | 15 | 应用框架 | papp.AppCtx 中间件能力增强 | 已有 TraceID 上下文基础，但 AppCtx 未暴露 TraceID/StartTime，HTTP 未接入 tracing middleware |
| Approved | 16 | 代码生成 | genCURD 支持多主键表 | 已确认发现联合主键后清空 PriCol，Update/Delete 不会生成 |
| Done | 18 | 基础设施 | 嵌套事务 | 为当前服务设计并实现 `pdb` 嵌套事务：业务传入事务 ID，内层 Begin 复用外层事务、任一层回滚整体回滚、兼容 defer rollback，并按 ID 自动取得事务查询对象 |

---

## 详细说明

### 12. pprof 性能分析

- **状态**：Approved（已规划）
- **背景**：项目未导入 `net/http/pprof` 或注册独立诊断端口，现有 `google/pprof` 间接依赖不能提供运行期 profile。
- **方案**：为服务增加独立、默认关闭的诊断 HTTP 端口，并只在显式配置开启时注册标准 pprof handlers；部署配置限制该端口仅对本机或受保护网络可见。补充 CLI 子命令封装常用 CPU、heap 和 goroutine profile 拉取，避免将诊断端点混入业务 HTTP 路由。
- **任务列表**：
  - 增加诊断端口及其开关、监听地址配置。
  - 在开发部署中验证 CPU、heap、goroutine profile。
  - 增加 CLI 拉取/打开 profile 的交互入口和关闭状态测试。
- **验收**：开启后可获取三类 profile；默认及生产限制配置下业务端口不暴露 pprof。

### 13. 持续监控

- **状态**：Approved（已规划；依赖任务 10 的字段约定）
- **背景**：Docker 部署已有 node-exporter、cAdvisor、Prometheus、Grafana、Loki 和 pm2 scrape 配置，但没有应用 `/metrics`、业务健康检查、数据库/Redis/RabbitMQ 指标或 `pgo` 的运维查询入口。
- **方案**：分层补齐应用 RED 指标（请求量、错误率、耗时）、依赖健康检查和业务状态指标；Prometheus 统一抓取，Grafana 展示并基于阈值告警。`pgo` CLI 提供只读健康概览、指标查询链接和常用诊断跳转，不复制 Grafana 的可视化能力。
- **任务列表**：
  - 定义应用、依赖和业务状态的最小指标与健康契约。
  - 接入指标端点、Prometheus scrape、仪表板和告警规则。
  - 增加 CLI 只读健康查询，覆盖服务不可达和部分依赖失败场景。
- **验收**：可观察每个服务的吞吐、延迟、错误率和健康状态；依赖异常触发可操作告警；CLI 能汇总当前状态。

### 14. genCURD inferServiceName 配置化

- **状态**：Approved（已规划）
- **背景**：`inferServiceName` 在 `cmd/pgo/tools/genCURD/genCURD_core.go` 内硬编码 `task`、`course` 等前缀和 `default` 回退；调用方无法覆盖规则。
- **方案**：为 genCURD 增加表名到服务名的显式映射参数，并支持从项目配置读取同一映射；显式参数优先于配置，未命中时保留当前前缀兼容规则并输出提示。开始生成前校验映射格式、重复表名和非法服务名，防止生成到意外目录。
- **任务列表**：
  - 扩展 CLI 参数与配置解析，并定义优先级和兼容回退。
  - 将服务名推断改为可注入规则，保留现有项目的生成结果。
  - 添加映射解析、优先级、非法输入和生成目录的单元测试。
- **验收**：指定表可生成到目标 service；无新配置时已有项目生成结果不变；非法映射在写文件前失败。

### 15. papp.AppCtx 中间件能力增强

- **状态**：Approved（已规划）
- **背景**：`AppCtx` 已匿名嵌入 `context.Context`，不缺失标准方法；`putil` 与 Kratos logger 已有 TraceID 传递，但 `AppCtx` 未暴露 TraceID、StartTime，HTTP 入口也未初始化 tracing middleware。
- **方案**：在入口 middleware 统一提取受信任的请求标识或生成新标识，并写入 context；`NewAppCtx` 从 Kratos trace、上下文回退值或新标识构造 `TraceID`，同时记录 `StartTime`。日志与响应使用同一标识；保持嵌入 `context.Context`，不另行复制其 API。
- **任务列表**：
  - 增加 HTTP/gRPC 请求标识与追踪上下文 middleware。
  - 扩展 AppCtx 的只读请求元数据和耗时日志辅助能力。
  - 测试 header 继承、自动生成、并发隔离、超时及日志关联。
- **验收**：每个请求都有稳定 TraceID 和开始时间；下游 context/日志/响应一致；标准 context 方法仍可直接使用。

### 16. genCURD 支持多主键表

- **状态**：Approved（已规划）
- **背景**：`newTable` 检测到多个主键后将 `PriCol` 置空；DAO、Proto、Service 模板据此移除主键相关的 Update/Delete 逻辑。
- **方案**：将生成器的单一 `PriCol` 模型扩展为有序主键列集合；Update/Delete 请求、服务转换和 DAO 条件都包含全部主键，并只以完整主键组合作为定位条件。保留无主键表的只读生成行为，并以 SQLite 和 MySQL 的联合主键样例回归。
- **任务列表**：
  - 重构表元数据和模板替换逻辑，使其接受主键列集合。
  - 更新 Proto、Service、DAO 生成结果和零值/类型校验。
  - 建立单主键、联合主键、无主键三组 fixture，并验证生成代码可编译及 SQL 条件完整。
- **验收**：联合主键表生成完整 Update/Delete；SQL 条件包含且仅包含全部主键；单主键和无主键行为不回归。

### 18. 嵌套事务

为当前服务设计并实现 `pkg/pdb` 的嵌套事务能力：

- 第一次 `Begin` 开启真实 GORM 事务并入栈登记；内层 `Begin` 只压栈计数，不新开事务
- 每层 `Commit` / `Rollback` 只更新本层状态；全部层都提交才真正提交，任一层回滚则整体回滚
- 兼容 `begin + defer rollback + commit` 惯用写法，commit 后再走到 defer 的 rollback 自动忽略
- 事务按业务传入的 ID 登记；`GetGormDB(id)` / `GetQueryTx(id)` 自动返回当前事务查询对象，已有 `GetQuery()` 调用无需改动
- 配套单测：嵌套 commit/rollback 组合矩阵、defer 场景、重复 commit/rollback 的告警分支

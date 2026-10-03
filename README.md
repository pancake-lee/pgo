# PGO

本仓库是个人习惯的一些封装（`pkg`）以及个人的小项目，主要用于学习和沉淀知识，并未打算成为一个"流行框架"。

## 项目组成

### 客户端（`cmd/pgo`）

一个 all-in-one 多模式运行的工具集，复用统一的核心逻辑，支持 CLI（Cobra）、CLI 交互式、GUI（Fyne）三种运行模式。

内置工具：
- **prettyCode**：统一代码中的分割线注释格式
- **psql**：连接 PostgreSQL 并执行 SQL，可嵌入自动化流程
- **sheet2mysql**：读取 APITable 表结构，生成 MySQL 建表 SQL
- **performance**：从组件导航页自动发现服务，执行登录或权限合并单档压测、固定阶梯升压，保存负载结果与测量窗口
- **CI Make**：交互式选择并执行 Makefile 目标
- **CI Init Project**：从当前仓库抽取项目基础骨架
- **CD Deploy**：首次部署容器及后续更新程序

### 服务端

| 服务 | 说明 |
|------|------|
| `userService` | 核心身份与权限管理，混合手写业务逻辑与自动生成 CRUD |
| `schoolService` | 换课客户端后端存储，纯代码生成 |
| `taskService` | 通用任务管理，纯代码生成 |
| `abandonCodeService` | 代码生成工具的演示与测试服务 |
| `ltblCallback` / `mtblCallback` | 外部多维表格 Webhook 回调，双向同步 |

### 框架能力

- **DB-First 全流程自动化**：SQL 定义 → `make gorm` 生成 ORM → `make curd` 生成 Proto/gRPC/HTTP/Service/Data 代码
- **多维表格驱动开发**（探索中）：APITable 前端建模 → sheet2mysql 生成建表 SQL → genCURD 生成后端代码 → 回调双向同步
- **微服务架构**：基于 Kratos，gRPC/HTTP 混合接口，Service → Data → DB 分层
- **丰富的组件封装**：`pkg/` 下统一封装 Log、Config、Redis、MySQL、RabbitMQ，以及微信生态（`pweixin`）、多维表格（`papitable`）等第三方集成

## 快速开始

```shell
# 安装依赖
make env

# 初始化数据库
make initDB

# 生成 ORM 代码
make gorm

# 生成 CURD 代码
make curd

# 构建
make build

# 查看所有可用目标
make help
```

### genCURD 表归属映射

`genCURD` 通过手写 Proto 中的 `pgo.tables` option 确定每张表属于哪个服务。例如，将 `task` 表归入 Task 服务：

```proto
syntax = "proto3";
package api;

import "pgo/options.proto";

service Task {
    option (pgo.tables) = "task";
}
```

使用流程：

1. 首次执行 `make curd`。未映射的表会生成到 `proto/z_defaultService.gen.proto`。
2. 在手写的服务 Proto 中导入 `pgo/options.proto`，并把 default service 中对应的 `option (pgo.tables)` 声明移入该服务。
3. 再次执行 `make curd`。生成器会将 Proto、Service 和 Data 代码迁移到目标服务。

一个服务可以重复声明 `pgo.tables` 以归属多张表。同一张表不能归属多个服务，映射中的表名也必须存在于当前数据库，否则生成器会在写入文件前报错。`z_*Service.gen.proto` 是生成文件，不要直接维护其中的映射。

完整的生成模板和可运行示例见 [`proto/abandonCode.proto`](./proto/abandonCode.proto) 与 [`internal/abandonCodeService/`](./internal/abandonCodeService/)。前者展示表归属、CRUD RPC、HTTP 路由、消息和字段生成规则，后者展示 Service 与 Data 层的模板结构。

### Docker Compose 组件导航

启动 `deploy/docker/docker-compose.yaml` 后，访问 `http://<部署主机>:20080`。导航页集中提供后端 API、pprof、RabbitMQ、Swagger UI、Prometheus、Grafana、Pyroscope、Alloy 和 cAdvisor 入口，并显示浏览器侧可达状态。`cmd/pgo/common` 解析同源的 `portal/services.json`，向 CLI 提供上述全部组件 URL；运行 `./bin/pgo performance login http://<部署主机>:20080` 后，性能工具只使用其中的 API URL、制造固定阶梯登录负载并保存 Vegeta 结果。服务指标和 profiling 直接在 Grafana、Pyroscope 中按压测时间范围查看。Alloy 将应用日志写入 Loki，并持续将 CPU、heap profile 写入 Pyroscope；goroutine、block、mutex 与 runtime trace 只能通过限时诊断接口采集。诊断端口不应暴露到公网。

## 文档索引

- [`CLAUDE.md`](./CLAUDE.md) / [`AGENTS.md`](./AGENTS.md)：AI 协作规则与工作模式
- [`docs/prd.md`](./docs/prd.md)：项目定位、范围与质量目标
- [`docs/tech.md`](./docs/tech.md)：架构、生成链路与技术边界
- [`docs/harness.md`](./docs/harness.md)：Harness 工程与文档入口
- [`docs/observability-data-map.md`](./docs/observability-data-map.md)：Grafana、Pyroscope、pprof 数据与代码来源速查
- [`docs/backlog.md`](./docs/backlog.md)：活跃需求池与任务交接
- [`docs/note.md`](./docs/note.md)：长期技术备忘与否决记录
- [`docs/eval/user-login-baseline.md`](./docs/eval/user-login-baseline.md)：多用户注册与登录性能评估
- [`docs/testing.md`](./docs/testing.md#权限合并性能练习)：权限数据准备、list/map 对照与图表检查
- [`docs/design/`](./docs/design/)：当前专题设计
- [`docs/archive/`](./docs/archive/)：已完成版本与历史归档

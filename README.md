# PGO

本仓库是个人习惯的一些封装（`pkg`）以及个人的小项目，主要用于学习和沉淀知识，并未打算成为一个"流行框架"。

## 项目组成

### 客户端（`cmd/pgo`）

一个 all-in-one 多模式运行的工具集，复用统一的核心逻辑，支持 CLI（Cobra）、CLI 交互式、GUI（Fyne）三种运行模式。

内置工具：
- **prettyCode**：统一代码中的分割线注释格式
- **psql**：连接 PostgreSQL 并执行 SQL，可嵌入自动化流程
- **sheet2mysql**：读取 APITable 表结构，生成 MySQL 建表 SQL
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

## 文档索引

- [`CLAUDE.md`](./CLAUDE.md)：AI 工作规则（编码规范、编译测试、禁忌清单）
- [`todo.md`](./todo.md)：当前周期任务清单
- [`docs/changelog.md`](./docs/changelog.md)：版本历史
- [`docs/backlog.md`](./docs/backlog.md)：全部需求池
- [`docs/note.md`](./docs/note.md)：活跃技术备忘
- [`docs/design/`](./docs/design/)：设计思路与方案记录

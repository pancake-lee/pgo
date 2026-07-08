# 目录结构设计

> 状态：已决策（采用方案 A），方案 B 为远期演进方向

## 问题背景

pgo 是一个以"定义 proto + sql → 生成一切"为核心的框架。开发者的日常工作流是：

1. 编写 proto 和 sql 作为项目入口定义
2. 运行代码生成工具（make gorm / make curd）
3. 在生成代码的基础上编写业务逻辑

而 proto 和 sql 作为最常翻阅的入口文件，在目录层级上不对等：

| 入口 | 原位置 | 层级 |
|------|--------|------|
| proto | `proto/` | 顶层，打开项目即见 |
| sql | `internal/pkg/db/*.sql` | 三层深，与生成的 model/query 挤在一起 |

每次要改表结构，需要在编辑器文件树中展开 `internal/` → `pkg/` → `db/` 才能看到 `.sql` 文件，与 proto 的操作体验差距明显。

## 方案分析

### 方案一：按"手写/生成"切分（已否决）

第一反应是把所有手写文件和生成文件分开放在不同目录：

```
proto/          (仅手写)
sql/            (仅手写)
gen/            (所有生成代码集中)
  ├── api/      (.pb.go)
  ├── proto/    (.gen.proto)
  ├── model/    (GORM model)
  └── query/    (GORM query)
```

**否决理由**：写业务代码时，开发者不关心文件是手写的还是生成的，只需要 `import ".../api"` 能找到类型。按"手写/生成"切分是框架作者的视角，加重了业务开发者的认知负担——要知道去哪找什么文件。

### 方案二：按 service 完全归拢（远期方向）

将一个 service 的所有文件放在同一目录下，proto 和 sql 作为入口文件直接放在 service 根：

```
internal/
├── userService/
│   ├── userService.proto
│   ├── userService.sql
│   ├── api/             ← protoc 生成
│   ├── db/              ← genGORM 生成 (model/query)
│   ├── data/            ← genCURD 生成 (dao)
│   └── service/         ← genCURD 生成 (svc/svr)
```

**优点**：
- 展开一个 service 目录就能看到所有相关文件
- 高内聚，适合 service 数量多的项目

**缺点**：
- 改动量巨大：protoc 需从一次全量编译改为 N 次按 service 编译；genGORM 需支持按表过滤分次生成；所有 Go import 路径全变
- 跨 service 的 proto 引用（`common.proto`、`error_reason.proto`）需要独立位置
- genGORM 底层 gentool 目前不支持按表过滤，需要外围改造
- `api.xxx` 跨服务调用时，需要知道类型属于哪个 service 的 api 包，编辑器自动补全效率下降

**最后一个缺点值得展开**：当前所有 proto 生成到同一个 `api/` 包，在代码中写 `api.` 后编辑器列出所有类型，不需要记住归属。如果拆成 `userService/api`、`schoolService/api`，跨服务调用时需要先 import 对应包，补全列表也被切碎。这个"flat api package"的设计在 service 数量不多时是很大的生产力优势。

### 方案三：入口定义提升 + 生成代码内聚到 internal/pkg（已采用）

第一步：把 `.sql` 文件从 `internal/pkg/db/` 提升到顶层 `sql/`，与 proto 平级。

第二步：把 `api/`（protoc 生成的 .pb.go）移入 `internal/pkg/api/`，将生成代码统一收进 `internal/pkg/` 下。

```
pgo/
├── proto/          ← 定义入口：API（手写）
├── sql/            ← 定义入口：数据库（手写）
├── internal/
│   ├── pkg/
│   │   ├── api/    ← protoc 生成
│   │   ├── db/     ← genGORM 生成 model/query + db.go
│   │   └── ...
│   └── xxxService/ ← 业务代码 + genCURD 生成
├── pkg/            ← 框架
└── cmd/            ← CLI
```

**优点**：
- proto 和 sql 作为两种项目入口定义，在顶层并列，认知层级统一
- 生成的代码（api、model、query）都在 `internal/pkg/` 下，顶层只留手写的入口文件
- `api/` 保留 flat package，跨服务补全不受影响
- import path 虽然变了，但路径语义更清晰：`internal/pkg/api` 一看就知道是生成的

**改动量**：
- SQL 移动：Makefile 路径变量、initProj.json、mysqlDiff.go 候选路径、sheet2mysql 默认路径
- API 移动：Makefile protoc 输出路径、~22 个 Go 文件的 import 路径、genCURD 模板 import、initProj import 重写逻辑

## 决策

采用**方案三**。理由：

1. 顶层只保留手写的入口定义（proto + sql），生成代码统一在 `internal/pkg/` 下
2. `api/` 的 flat package 保持在同一个包内，编辑器补全不受影响
3. `internal/` 内部的手写/生成混放通过 `z_` 前缀区分已经够用，不需要进一步拆分
4. 改动成本可控，不影响目标项目的目录结构

## 改动记录

### 第一次：SQL 提升到根目录

| 文件 | 改动 |
|------|------|
| `internal/pkg/db/*.sql` → `sql/` | SQL 文件整体迁移 |
| `Makefile` | `dbCodePath` 默认值、`initDB`/`reInitDB` 路径 |
| `pkg/pdb/mysqlDiff.go` | `resolveSchemaSQLDir()` 候选路径增加 `./sql` |
| `cmd/pgo/tools/sheet2mysql/sheet2mysql.go` | 默认输出目录改为 `./sql/` |
| `deploy/initProj.json` | SQL 文件复制源路径 |
| `deploy/deploy.json` | 部署时的 SQL glob 路径 |
| `deploy/initReadme.md` | 初始化指引中的目录引用 |

### 第二次：api/ 移入 internal/pkg/

| 文件 | 改动 |
|------|------|
| `api/` → `internal/pkg/api/` | protoc 生成的 .pb.go 整体迁移 |
| `Makefile` | protoc 的 4 个 `--*_out` 输出路径 |
| ~22 个 `.go` 文件 | import `pgo/api` → `pgo/internal/pkg/api` |
| `cmd/pgo/devops/ci_initProj.go` | import 重写逻辑：先匹配 `internal/pkg/api` 再匹配 `internal` |
| `internal/abandonCodeService/service/` | genCURD 模板中的 import 路径 |

## 未来演进

当 service 数量增长到 10+ 时，可以考虑向方案二演进。演进前需解决的前置条件：

- gentool 支持按表名过滤生成，或 pgo 封装层实现表→service 路由
- protoc 支持按 service 目录的增量编译策略
- 确定跨 service 共享 proto 的放置规范
- 评估 `api.xxx` flat package 拆散后对开发效率的实际影响（可在拆分前用子包别名模拟体验）

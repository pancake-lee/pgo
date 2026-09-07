# PGO 仓库全面评估

## 摘要

- **综合得分**：5.6 / 10，未通过（低于 6 分，且正确性维度为 5.0）。
- **范围**：319 个 Go 文件、53 个包、17 个测试文件；涵盖 CLI、代码生成、基础设施封装、服务样例、构建与文档承诺。
- **结论**：架构边界、文档和正式构建具备可用基础；但全量测试门槛失效、生成失败可能破坏现有产物、构建元数据错误，使 P0 工具链尚不能被稳定地作为回归基准。

## 评分

### 正确性：5.0

得分点：

- `GOTOOLCHAIN=local make build` 成功，所有程序均输出到 `bin/`，符合技术文档的产物边界。
- 受影响的 `diagnostics`、`prettyCode`、`pconfig`、`pdb`、`putil` 包测试通过。

失分点：

- `GOTOOLCHAIN=local go test ./...` 失败。`pkg/papp.TestJwt` 可重复报 `auth failed`；测试将 token 放入 Kratos 的 context key，但 `GetTokenFromCtx` 读取项目私有 key，测试未覆盖实际中间件写入路径。
- `pkg/plogger/warper.go:37` 的动态 `fmt.Errorf(errMsg)` 被 `go vet` 判为非固定格式串，`go test` 与 `go vet ./...` 均因此失败。
- `cmd/pgo/tools/genCURD/genCURD_core.go:176-193` 先调用删除生成文件的 `rmAllGenFile`，再初始化数据库；数据库连接或读取失败时，无法满足 PRD 中“生成失败不得留下半成品”的要求。

### 健壮性：5.5

得分点：

- `papp` 提供请求 ID、Prometheus 指标、健康检查和受限格式的请求 ID 校验；数据库、Redis、RabbitMQ 的健康检查可按已初始化依赖启用。
- `genGORM` 对空 DSN、未知数据库类型和必要输出参数返回错误。

失分点：

- `pkg/papitable`、`pkg/predis`、`pkg/pweixin` 及用户服务测试依赖未提交的 `.local/my-config.yaml`、外部服务或数据库，缺失时使用 `MustInitConfig` 直接 panic；全量验证无法区分可离线运行的单元测试与环境测试。
- 用户服务测试未初始化 `pdb`，调用 generated query 时将 nil DB 传入 GORM，触发空指针 panic。

### 可维护性：6.0

得分点：

- `README.md`、PRD、技术文档和编码规范明确了项目定位、包边界、生成链路与构建入口。
- CLI、`pkg/`、`internal/`、`proto/`、`sql/` 的顶层职责与技术文档一致；生成代码有明确标记，便于审阅时排除。

失分点：

- 53 个包中仅 12 个具有测试文件；`genCURD`、`genGORM`、`sheet2mysql` 和多数服务实现没有自动化验证。
- 非生成代码中仍有 15 个 TODO/FIXME，集中在表格同步、DDL 转换、CLI 初始化和消息重试等功能路径；部分问题未在活跃 backlog 中建立对应记录。
- `GOTOOLCHAIN=local go mod tidy -diff` 报告 `go.mod` / `go.sum` 漂移，依赖声明的直接/间接层级与实际导入未保持一致。

### 简洁性：6.0

得分点：

- 生成层将 API、DAO、服务样例的重复样板代码集中，代码库的领域实现与生成代码基本隔离。
- `papp` 的观测与健康检查能力集中在一个基础包，服务侧无需重复接线。

失分点：

- 代码生成流程同时依赖 Makefile、安装到环境的 `pgo` 命令和生成器内嵌的 `make api`，执行链路跨多个隐式入口。
- 当前仓库同时保留多个数据库驱动/SQLite 实现的直接与间接依赖，模块整理结果显示声明与实际使用存在冗余或漂移。

### 功能完整性：5.5

得分点：

- CLI 覆盖项目初始化、代码生成、数据库辅助、格式化和诊断等 README 所述主要入口；服务端具有可运行的 HTTP/gRPC 分层和生成样例。
- 当前 backlog 已明确记录服务名映射和联合主键两项生成器缺口，问题可追溯。

失分点：

- P0 的“代码生成结果可编译且可验证”本轮无法由自动化回归证明：生成器核心包没有测试；正式 `gorm` 目标会删除数据库及生成目录，不能作为无副作用检查运行。
- 生成器仍明确将联合主键降级为无主键行为，且服务名规则为硬编码前缀；两项均处于已规划但未实现状态。

### 一致性：6.0

得分点：

- 文档、Makefile 和源码对 `bin/` 作为可执行产物目录的约定一致；实际构建已验证。
- 服务端生成代码的 CRUD 命名和分层形式在多个示例服务间保持一致。

失分点：

- 技术文档规定构建与测试采用 `GOTOOLCHAIN=local`，但 Makefile 的 Go 命令未显式传入该环境约束。
- `Makefile` 的 `build` 目标在实际执行时传入 `main.commit=` 和 `main.date=` 空值：其中 `$(git rev-parse HEAD)` 与 `$(date ...)` 被 Make 当作变量引用，和 `cli` 目标的 shell 写法不一致。
- 未发现 CI 配置，仓库缺少一处持续执行文档所列构建、测试和静态检查的事实来源。

### 开发者可用性：5.0

得分点：

- README 提供从环境、数据库、生成到构建的最短路径；`make help` 可列出主要目标。
- 配置加载过程会打印所尝试的配置文件，外部依赖缺失时有可观察的错误线索。

失分点：

- README 快速开始无法在干净工作区完成全量验证：多个测试指向不存在的 `.local/my-config.yaml`，但仓库仅提交 `common.yaml` 和 `courseSwap.yaml`。
- `make env` 包含系统包管理、全局安装和 `go mod tidy`，其前置条件与平台差异未形成可自动验证的流程；本轮也未发现 CI 兜底。

## 执行证据与边界

- 通过：`GOTOOLCHAIN=local make build`。
- 失败：`GOTOOLCHAIN=local go test ./...`、`GOTOOLCHAIN=local go vet ./...`。
- 通过：`diagnostics`、`prettyCode`、`pconfig`、`pdb`、`putil` 的定向测试。
- 失败：`papp.TestJwt`、`plogger` 的 vet、以及依赖本地配置/服务的 `papitable`、`predis`、`pweixin`、用户服务测试。
- `GOTOOLCHAIN=local go mod tidy -diff` 发现模块元数据差异；未写入文件。
- 未执行 `make gorm`、`make curd`：前者会执行 `DROP DATABASE` 并删除生成目录，属于破坏性操作；本报告不把未执行验证记作通过。
- 未启动服务或后台进程；工作区在评估结束时仅包含报告与 backlog 文档变更。

## 进入规划的发现

- 见 backlog 17–21。它们仅记录现象、影响与严重程度，未包含实现方案。

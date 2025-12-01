# PGO

本仓库是个人习惯的一些封装(pkg)以及个人的小项目。
主要用于学习/练习/沉淀知识，并未打算称为一个“流行框架”

## AI PROMPT

### **Project PGO Development Context & Guidelines**

**1. 开发模式 (Development Workflow)**

* **DB/Model First**: 项目以数据库表结构（GORM Model）为核心。
  * 编写好 `internal/pkg/db`的表定义后使用 `make gorm`生成orm代码
* **代码生成 (Code Generation)**:
  * 使用自定义工具 `genCURD` (`go run ./tools/genCURD/`)。
  * 该工具通过反射 (`reflect`) 读取数据库表结构和索引信息。
  * 自动生成内容：Proto 定义 (`proto/`)、gRPC/HTTP 桩代码 (`api/`)、Service 层基础 CRUD (`z_svc_*.gen.go`)、Data 层基础 CRUD (`z_dao_*.gen.go`)。
  * **注意**: 以 `z_` 开头 `gen.go`结尾的文件为自动生成，**禁止手动修改**。
* **自定义逻辑 (Custom Logic)**:
  * 在 `proto/` 中定义非 CRUD 的额外 RPC 接口。
  * 在 `internal/<service>/service/` 中实现业务逻辑。
  * 在 `internal/<service>/data/` 中实现复杂的数据库查询。

**2. 分层架构与职责 (Architecture & Responsibilities)**

* **Service 层 (`internal/<service>/service`)**:
  * **职责**: 处理业务逻辑，参数校验，调用 Data 层，模型转换 (Model -> Proto)。
  * **风格**: 保持逻辑清晰，尽量不直接操作 DB，而是通过 Data 层接口。
* **Data 层 (`internal/<service>/data`)**:
  * **职责**: 封装所有数据库操作 (DAO 模式)。
  * **风格**:
    * **文件拆分**: 按表/实体拆分文件 (e.g., `dao_UserRole.go`, `dao_UserRoleAssoc.go`)，避免大杂烩。
    * **返回值**: 尽量返回完整的 Model 对象指针 (`*model.User`) 或列表 (`[]*model.User`)，而非仅仅返回 ID，以便上层灵活使用。
    * **Context**: 数据库操作需传递 `context.Context` 以支持链路追踪或超时控制。

**3. 编码规范 (Coding Conventions)**

* **变量命名**:
  * 列表/切片后缀使用 `List` (e.g., `userRoleList`, `permissionList`)。
  * Map 结构后缀使用 `Map` (e.g., `roleIDMap`, `permissionMap`)。
* **测试 (Testing)**:
  * 编写集成测试 (Integration Tests) 验证 Service 层逻辑。
  * **数据清理**: 使用 `defer` 配合清理函数 (e.g., `defer testDelUser(...)`) 移除测试数据。
  * **禁忌**: 禁止在测试中直接修改表结构，这会破坏表结构和其他测试的运行。表结构的差异应该正常报错，有利于提醒本次更新涉及到数据库结构更新，升级服务器时需要注意到。

**4. 工具链细节 (Tooling Insights)**

* **genCURD**:
  * 目前已支持通过 GORM `Migrator().GetIndexes()` 识别数据库索引。
  * 未来扩展方向：根据识别到的唯一索引 (`UniqueIndex`) 自动生成 `GetBy<IndexColumn>` 等查询方法。

## [commitlint](`https://github.com/conventional-changelog/commitlint`)

| prefix   | desc       |
| -------- | ---------- |
| build    | 构建相关   |
| chore    | 杂项       |
| ci       | CI/CD 相关 |
| docs     | 文档       |
| feat     | 功能       |
| fix      | 修复       |
| perf     | 性能       |
| refactor | 重构       |
| revert   | 回退       |
| style    | 代码风格   |
| test     | 测试       |
| gen      | 生成代码   |
| improve  | 优化代码   |
| tidy     | 整理、清理 |

## [semver](https://semver.org/lang/zh-CN/)

版本格式：主版本号.次版本号.修订号，版本号递增规则如下：

- 主版本号：当你做了不兼容的 API 修改，
- 次版本号：当你做了向下兼容的功能性新增，
- 修订号：当你做了向下兼容的问题修正。
  先行版本号及版本编译信息可以加到“主版本号.次版本号.修订号”的后面，作为延伸。

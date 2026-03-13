# PGO

本仓库是个人习惯的一些封装(pkg)以及个人的小项目。
主要用于学习/练习/沉淀知识，并未打算称为一个“流行框架”

## AI小规则

- 阅读readme然后等待指令
  - 每次新的对话，我都会让你先阅读readme，了解项目当前情况，你只需要回复收到
- 请总结变更到readme
  - 每次完成一个需求点，我会请你总结“变更内容”到当前文档(readme.md)
  - 你应该按照当前文档的标题结构，分类修改或添加内容
  - 不要单纯追加内容，导致文档无限增长
- 不修改
  - 当我说“不修改”时，则当前内容只希望你分析问题/提供方案，但不要修改代码
- 代码风格以“手动修改代码”为最高优先级
  - AI在实现需求前，需要先阅读当前相关代码，优先复用你已经手动沉淀的命名、分层、注释和参数组织方式
  - 当手动代码与历史实现冲突时，以最新手动代码风格为准
- 本文的TODO中，已完成的功能从TODO删掉，功能适当写到合适的地方，而不是在TODO中描述实现的功能

## Features

### 客户端

- 基础
  - all in one = 多模式运行
    - 复用统一的核心逻辑，封装在`client/common/tool_entrypoint.go`
    - 支持 CLI (Cobra) 命令行输入参数
    - 支持 CLI (Interactive) 命令行进入程序，持续交互输入参数
    - 支持 GUI (Fyne/Windows)
  - 命令行入口统一为 Cobra：`runCli -> rootCmd.Execute()`，可直接提供 `help` 指引。
  - 运行行为约定：命令行无参数进入持续交互菜单；Windows 双击（无参数）保持 GUI；`client.exe cli` 兼容旧入口。
- 工具
  - prettyCode 美化代码
    - 统一规范代码中的分割线注释格式（如 `// -[*50]`）
  - psql：连接 PostgreSQL 并执行 SQL 语句或 SQL 文件
    - 自己实现的重要原因是官方psql不能参数输入密码，不方便嵌入到自动化流程中
    - 后来才发现可以环境变量输入，不过已经实现好了，就当减少一个依赖吧
  - sheet2mysql：读取 APITable 表结构并生成 MySQL 建表 SQL
    - 支持多维表格前端建模（APITable），自动转换表名与字段名为拼音标识符
    - 参数支持 config 回填：当 url/spaceId/token 参数为空时，从配置文件中自动读取
    - 生成逻辑简化：始终生成 `DROP TABLE IF EXISTS`，始终跳过计算列（公式/引用/按钮）
    - 输出到指定文件夹，文件名由表名自动生成（格式：`<resolvedTableName>.sql`）
- sdk
  - swagger目录包含了后端接口的生成代码，无需手搓http请求
  - 参考makefile中api-cli命令，可以通过swagger生成其他语言的代码
- CI
  - Make
    - 读取makefile的命令和备注信息，提供执行的功能
    - 则make命令不再是死记硬背，而是命令行交互选择功能
    - 参数可以交互式输入，然后记录到缓存文件，下次询问更改或使用旧值
      - 如生成orm代码所需要的数据库host/port/user/pass等
  - Init Project
    - 提供`initProj`初始化能力：从当前仓库抽取新项目基础骨架到目标目录。
    - 拷贝列表配置化：由`deploy/initProj.json`维护`files`与`exclude`，无需改代码即可调整初始化内容。
    - 安全策略：不自动删除目标目录内容；若检测到目标文件冲突则打印冲突列表并中断，交由用户手动处理。
    - 模块初始化：不拷贝`go.mod/go.sum`，在目标目录执行`go mod init <目标目录名>`与`go mod tidy`。
    - 拷贝后文本修正：按目标目录名替换导入前缀，包含`github.com/pancake-lee/pgo/internal` -> `<目标模块>/internal`与`github.com/pancake-lee/pgo/api` -> `<目标模块>/api`。
- CD
  - 提供首次部署容器的能力
    - 由./deploy/deploy.json定义部署需要的文件列表，kv记录src和远程dst路径映射
    - 支持可选的`exclude`列表，用于按相对路径排除不需要上传的文件/目录（支持目录前缀和glob）
    - 指定ssh目标机器，以及部署根目录，参数缓存到~/pgo/cache.json
    - 拷贝文件到远程目录，遇到已存在文件，检测md5一致则跳过，不一致则提示冲突，中断部署
      - 这意味着用户一旦自己修改了部署细节，后续需要自行维护，以免工具覆盖了用户的修改
    - 用户选择自动运行整套服务，还是手动运行
      - 运行命令为`docker-compose -f pgo.yaml up -d`
      - 用户选择手动，则仅打印提示用户这个命令，选择自动则通过ssh直接执行命令
  - 也提供更新程序的能力，包括更新数据库结构和程序
  - 数据库结构不能drop，仅代码层面废弃即可，实际数据清理需要用户自行操作
  - 不处理科学上网的问题，但让用户有“自行下载并把所需文件放到指定目录下”的方法
    - 这是很后面的优化提升了，可能是docker-compose或者各个docker镜像的本地安装

### 服务

- 用户服务 (`userService`)
  - 定位: 核心身份与权限管理服务。
  - 功能: 提供用户登录 (`Login`)、个人信息修改、权限查询 (`GetUserPermissions`)。
  - 特性: 混合模式，同时包含手写业务逻辑与自动生成的 CRUD 接口。

- 学校服务 (`schoolService`)
  - 定位: 换课客户端 (`CourseSwap`) 的后端存储支撑。
  - 功能: 管理换课申请单 (`CourseSwapRequest`)，记录源/目标教师与课程的调换详情。
  - 特性: 纯代码生成服务，展示 `genCURD` 对新业务实体的快速支持。

- 任务服务 (`taskService`)
  - 定位: 通用的任务管理服务。
  - 功能: 管理任务 (`Task`) 的层级关系、状态流转及时间规划 (起止/估时)。
  - 特性: 纯代码生成服务，提供标准的任务数据操作接口。

- 示例服务 (`abandonCodeService`)
  - 定位: 代码生成工具的演示与测试服务。
  - 功能: 对 `abandon_code` 表提供标准的增删改查能力。
  - 特性: 模板服务，用于验证工具链生成的代码结构的正确性。

- 回调处理 (`ltblCallback` / `mtblCallback`)
  - 定位: 外部多维表格系统的集成模块。
  - 功能: 接收并处理 Webhook 回调，实现本地数据与外部表格 (如飞书/维格表) 的双向同步。
  - 特性: 集成服务，专注于外部生态连接。

### 框架

- 全流程自动化开发 (DB-First)
  - SQL定义优先: 编写 SQL 脚本后，通过 `make gorm` 自动生成 GORM 模型代码。
  - 代码生成器 (`tools/genCURD`): 根据数据库 Schema 自动生成 Proto 定义、gRPC/HTTP 桩代码、Service 业务层及 Data 数据层的基础 CRUD 代码。
  - SDK 自动构建: 支持 `make api-cli` 基于 OpenAPI 规范自动生成 Go 语言客户端 SDK。

- 多维表格驱动的混合开发模式（探索中）
  - 目标: 让业务侧优先在多维表格完成建模，再由后端工程化承接，形成“表格前端 + MySQL 后端 + 可扩展 API”的落地路径。
  - 已实现:
    - 基于 MySQL 表结构自动生成 ORM 与基础 CRUD 代码（`make gorm` + `genCURD`）。
    - 已沉淀多维表格集成能力（`pkg/papitable`）与回调同步模块（`ltblCallback` / `mtblCallback`）。
  - TODO:
    - 增加“多维表格结构 -> MySQL 表结构”生成工具，打通反向建模链路。
    - 补齐双向同步中的冲突处理、幂等与回环抑制策略，降低数据一致性风险。
    - 逐步适配 APITable 之外的平台（如飞书/企微），降低单平台绑定风险。

- 微服务架构支持
  - 基于 Kratos 框架，提供标准化的 gRPC/HTTP 混合接口支持。
  - 清晰的分层架构：`Service` (业务逻辑) -> `Data` (数据访问) -> `DB`，生成的代码 (`z_*.gen.go`) 与自定义逻辑分离。

- 丰富的组件封装 (`pkg`)
  - 基础设施: 统一封装了 Log, Config, Redis, MySQL, RabbitMQ 等基础组件。
  - 集成能力: 内置微信生态 (`pweixin`) 及多维表格 (`papitable`) 等第三方服务集成。

- DevOps 友好
  - 完善的 Makefile 支持：涵盖环境安装 (`env`)、代码生成 (`api`, `gorm`, `curd`)、构建 (`build`) 及 部署准备。

## TODO

- 多维表格驱动系统化落地
  - 新增“多维表格结构 -> MySQL 表结构”生成工具（字段类型映射、索引/约束映射、增量变更策略）。
  - 定义并实现双向同步冲突规则（双端同时修改优先级、版本号/时间戳策略、回环标记规范）。
  - 建立同步可观测性（重试队列、失败补偿、告警与审计日志），支撑后续跨行业推广。
  - 抽象多平台适配层，统一 APITable/飞书/企微等多维表格接入接口。

- 多维表格的双向同步中，TEMP标记应该区分两个方向，否则有如下BUG
  - 多维表格不开启自动化的回调webhook，而是放弃同步，或者按钮触发同步
  - 本地第1次修改数据，回调设置TEMP，然后修改多维表格数据
  - 本地第2次修改数据，回调中发现当前TEMP状态，以为是多维表格的回环同步，中断逻辑
- 做一个案例，把排查BUG的套路走一遍
  - 编写一个测试接口，该接口本质上包含3个内部接口调用
  - 这些代码内有包含延时/日志/随机错误，也要包含并发，让日志产生“交错”
  - 报错时需要在response中返回一个key，可能是RequestID
    - 为了方便调用者上报，要考虑常见浏览器中获取这个key是否足够便捷
  - 根据这个key，可以很容易地在日志平台找到所有相关日志，还原业务过程
  - 进一步需要探究对于“生产环境”，一个接口应该打印哪些数据
    - 性能和排查信息之间的平衡在哪里
    - 考虑缓存日志在内存，不出错则不打印，只有warn和error等级才全部输出的策略
  - 进一步可以提供jeager的链路追踪，确认Grpc的接口调用链条
  - 进一步可以提供pprof信息，确认接口内部代码的函数调用链条
- 更多的监控
  - 除了上面日志平台提供“事后排查”，要提供更多“持续监控”
  - 接口调用频率，耗时，成功率等等数据，还有硬件数据
  - 各服务健康状况，包括中间件和服务进程和一些自动业务的状态
  - 控的方面更加倾向于用client来承载，同样使用corba交互操作
    - corba似乎更多是命令行执行，但不是连续交互的性质，可能要换库
- 对于其他项目想要采用该项目的开发模式
  - bootCheck不要依赖orm，才能用于其他项目

## AI PROMPT

### Project PGO Development Context & Guidelines

AI 助手启动提示词 (Initial Prompt for AI Assistant)：
你好！请先读取项目根目录下的 `README.md` 文件，完整了解 Project PGO 的项目背景、开发模式、分层架构、编码规范及工具链细节。
当我完成一个需求的开发，我会向你发送 '总结到项目'，你可以选择性把有用的内容回写到根目录下的 `README.md` 中，注意必须是简洁的高度总结，以免篇幅过大。
注意：本条消息不需要你进行代码编写或回答具体问题。请仅回复 '收到 (Received)' 以确认你已加载项目上下文。具体的开发任务我将在下一条消息中给出。

#### 1. 开发模式 (Development Workflow)

- DB/Model First: 项目以数据库表结构（GORM Model）为核心。
  - 编写好 `internal/pkg/db`的表定义后使用 `make gorm`生成orm代码
- 代码生成 (Code Generation):
  - 使用自定义工具 `genCURD` (`go run ./tools/genCURD/`)。
  - 该工具通过反射 (`reflect`) 读取数据库表结构和索引信息。
  - 自动生成内容：Proto 定义 (`proto/`)、gRPC/HTTP 桩代码 (`api/`)、Service 层基础 CRUD (`z_svc_*.gen.go`)、Data 层基础 CRUD (`z_dao_*.gen.go`)。
  - 注意: 以 `z_` 开头 `gen.go`结尾的文件为自动生成，禁止手动修改。
- 自定义逻辑 (Custom Logic):
  - 在 `proto/` 中定义非 CRUD 的额外 RPC 接口。
  - 在 `internal/<service>/service/` 中实现业务逻辑。
  - 在 `internal/<service>/data/` 中实现复杂的数据库查询。

#### 2. 分层架构与职责 (Architecture & Responsibilities)

- Service 层 (`internal/<service>/service`):
  - 职责: 处理业务逻辑，参数校验，调用 Data 层，模型转换 (Model -> Proto)。
  - 风格: 保持逻辑清晰，尽量不直接操作 DB，而是通过 Data 层接口。
- Data 层 (`internal/<service>/data`):
  - 职责: 封装所有数据库操作 (DAO 模式)。
  - 风格:
    - 文件拆分: 按表/实体拆分文件 (e.g., `dao_UserRole.go`, `dao_UserRoleAssoc.go`)，避免大杂烩。
    - 返回值: 尽量返回完整的 Model 对象指针 (`*model.User`) 或列表 (`[]*model.User`)，而非仅仅返回 ID，以便上层灵活使用。
    - Context: 数据库操作需传递 `context.Context` 以支持链路追踪或超时控制。

#### 3. 编码规范 (Coding Conventions)

- 变量命名:
  - 列表/切片后缀使用 `List` (e.g., `userRoleList`, `permissionList`)。
  - Map 结构后缀使用 `Map` (e.g., `roleIDMap`, `permissionMap`)。
- Go 代码格式化:
  - 代码逻辑修改时不需要纠结缩进/对齐等格式问题，先直接改逻辑。
  - 改完后统一使用 `gofmt -w <file>` 处理格式，不用手写脚本修改格式。
- 测试 (Testing):
  - 编写集成测试 (Integration Tests) 验证 Service 层逻辑。
  - 数据清理: 使用 `defer` 配合清理函数 (e.g., `defer testDelUser(...)`) 移除测试数据。
  - 禁忌: 禁止在测试中直接修改表结构，这会破坏表结构和其他测试的运行。表结构的差异应该正常报错，有利于提醒本次更新涉及到数据库结构更新，升级服务器时需要注意到。
- 其他
  - 个人不喜欢 `if`中使用 `;`的写法，很容易造成长代码，如 `if d,ok:=data["k"]; ok`
  - 入口函数命名: 习惯将 `main` 方法写到 `internal/<module>` 对应模块的同名代码文件中（例如 `internal/bootCheck/bootCheck.go`），而不是 `cmd/` 下。
  - 类型抽象: 非复杂场景优先用基础类型组合，避免为简单函数签名额外定义 `type`；仅在复用明显或封装语义明确时再抽象。

- 客户端工具开发规则（以当前手动代码为准）
  - 入口统一为“单例暴露”模式：
    - 工具内部核心执行函数统一为 `Run(values common.ParamMap) error`。
    - 每个工具导出 `Entrypoint`（`common.NewToolEntrypoint(...)`）。
    - 外部统一调用 `Entrypoint` 的3个调用方法
      - `RunInteractive()`：只负责交互采参（可带缓存）。
      - `NewCobraCommand()`：只负责命令注册与参数解析。
      - `Run(options)`：只负责核心业务逻辑并返回 `error`。
  - 参数统一注册，不重复写两套参数：
    - 统一维护参数列表（如 `[]common.ParamItem`），字段包含 `Name/Prompt/Usage/Default`。
    - 通过公共方法同时服务 Cobra 与交互输入（如 `RegParamToCobra`、`ParseParamFromCobra`、`GetCachedParamMap`）。
  - 参数命名与转换分层：
    - 参数名常量集中定义（如 `paramNameXXX` + `cacheKeyPrefix`）。
    - 通过单独转换函数（如 `convParamToRunOpt`）把参数映射为运行结构体（`RunOptions`）。
  - 参数 config 回填设计：
    - 特定工具可添加 `config` 参数（指向配置文件路径），当某些参数（如 token、baseUrl、spaceId）为空时，从配置文件中自动读取。
    - 配置读取示例：检查 `configPath != ""` 且参数为空，则调用 `pconfig.InitConfig(configPath)` 再用 `pconfig.GetStringD()` 读指定 key。
    - 回填优先级：CLI 参数 > config 文件 > 代码默认值（如默认 `baseUrl="https://aitable.ai"`）。
  - 代码组织风格：
    - 关键区域使用 `// --------------------------------------------------` 分段。
    - 注释偏中文、直述意图，先说明“职责”，再写“实现细节”。
    - 参数列表拆分时优先按功能组（如认证参数、输出参数），保持 `ParamItem` 声明的可读性。

#### 4. 工具链细节 (Tooling Insights)

- genCURD:
  - 目前已支持通过 GORM `Migrator().GetIndexes()` 识别数据库索引。
  - 未来扩展方向：根据识别到的唯一索引 (`UniqueIndex`) 自动生成 `GetBy<IndexColumn>` 等查询方法。

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

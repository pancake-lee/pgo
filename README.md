# PGO

本仓库是个人习惯的一些封装(pkg)以及个人的小项目。
主要用于学习/练习/沉淀知识，并未打算称为一个“流行框架”

## Go 编码习惯

### 命名规范

- **列表/切片**: 后缀 `List` (e.g., `userRoleList`, `permissionList`)
- **Map 结构**: 后缀 `Map` (e.g., `roleIDMap`, `permissionMap`)

### 代码格式化

- 逻辑修改时不纠结缩进/对齐，改完后统一用 `gofmt -w <file>` 处理

### 测试规范

- 编写集成测试验证 Service 层逻辑
- 使用 `defer` + 清理函数移除测试数据 (e.g., `defer testDelUser(...)`)
- **禁忌**: 禁止在测试中修改表结构，差异应正常报错以提醒升级注意

### 个人习惯

- 避免 `if` 中使用 `;` (如 `if d,ok:=data["k"]; ok`)，易造成长代码
- 入口函数放在 `internal/<module>/<module>.go`，而非 `cmd/`
- 非复杂场景优先用基础类型组合，仅在复用明显或封装语义明确时抽象 `type`
- **接口代理模式**: 基础类通过接口代理支持子类覆盖，子类初始化后调用 `BindProvider(self)`

## Features

### 客户端

- 基础
  - all in one = 多模式运行
    - 复用统一的核心逻辑，封装在`client/common/tool_entrypoint.go`
    - 支持 CLI (Cobra) 命令行输入参数
    - 支持 CLI (Interactive) 命令行进入程序，持续交互输入参数
    - 支持 GUI (Fyne/Windows)
      - 左侧：功能列表
      - 中间上半部分：固定参数输入
        - 字符串输入
        - 日期选择
        - 下拉选择
      - 中间下半部分：（如果有）展示当前已存在数据
        - 提供字符串列表，字符串由业务代码自己编解码，底层直接存储/展示字符串即可
        - 提供两个按钮，修改和删除，逻辑由业务代码传入callback
      - 右侧上半部分：展示本程序日志
      - 右侧下半部分：本程序输出展示
        - 提供字符串列表，字符串由业务代码自己编解码，底层直接存储/展示字符串即可
        - 提供两个按钮
          - 复制：点击后复制到粘贴板
          - 确认：逻辑由业务代码传入callback
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
  - 代码生成器 (`tools/genCURD`): 根据数据库 Schema 自动生成 Proto 定义、gRPC/HTTP 桩代码、Service 业务层、Data 数据层基础 CRUD 代码，以及服务入口 main 文件（基于 `abandonCode` 模板替换生成）。
  - SDK 自动构建: 支持 `make api-cli` 基于 OpenAPI 规范自动生成 Go 语言客户端 SDK。

- 多维表格驱动的混合开发模式（探索中）
  - 目标: 让业务侧优先在多维表格完成建模，再由后端工程化承接，形成“表格前端 + MySQL 后端 + 可扩展 API”的落地路径。
  - 已实现:
    - 基于 MySQL 表结构自动生成 ORM 与基础 CRUD 代码（`make gorm` + `genCURD`）。
    - 已沉淀多维表格集成能力（`pkg/papitable`）与回调同步模块（`ltblCallback` / `mtblCallback`）。
    - 优化 `pkg/papitable`：`GetCols` 添加 1 分钟本地缓存，新增/删除列操作会使缓存失效；`ParseMultiOptionValue` 兼容 `[]interface{}`（`[]any`）类型解析，减少 API 响应处理错误。
    - BaseDataProvider 接口代理设计：所有 DataProvider 方法调用通过代理分发，子类可覆盖任意方法实现自定义逻辑（调用 `BindProvider()` 绑定子类实例）。

    - 实践流程（从 APITable 到双向同步，精简版）：
      - 在 APITable 设计业务表：确认 `DatasheetID`、`TableName`、首列（`FirstCol`）、列列表（`ColList`）与本地主键（`PrimaryCol`）。
      - 使用 `sheet2mysql` 生成建表 SQL（输出 `<resolvedTableName>.sql`，默认生成 `DROP TABLE IF EXISTS`，跳过计算列），手工调整字段类型、索引、默认值与约束。
      - 导入或执行 SQL 后，运行 `make gorm` / `pgo gorm/curd`（底层调用 `tools/genCURD`）生成 GORM model、`proto/`、API 桩与基础 CRUD 代码。
      - 补充 Data 层实现：编写 `data.xxx.go`，实现 DO/DAO、`UpdateMtblInfo`，并提供 `DAOWrapper`（把 DO/DAO 封装为同步层期望的 `any`）。
      - 实现 Service 层：在 `internal/<service>/service/` 完成生成的 `Unimplemented*Server` 的具体逻辑，并在 `papp.RunKratosApp` 注册该服务。
      - 回调与同步组件：
        - `mtblCallback`：为 APITable 提供 HTTP 回调入口，解析行数据并通过 `NewDO` 转换为本地 DO，再通过 RabbitMQ/队列投递到本地消费流程。
        - `ltblCallback`：监听本地数据库变更，转换为 APITable 行更新/新增请求并调用 APITable API（注意仓库默认实现偏向 mtbl->ltbl 单向，同步到 APITable 的逻辑可能需要针对表结构扩展）。
      - 同步锚点与防环策略：
        - 使用本地数据库的 `PrimaryCol` 作为双方同步的锚点，避免使用 APITable 行 id 作为主同步键。
        - `TEMP` / 回环标记应区分方向（local→remote 与 remote→local），并实现幂等、冲突解决（时间戳/版本号）与回环抑制。
      - 数据适配：优先使用 `BaseDataProvider` 实现通用字段的解析/序列化，需要特殊处理时继承或实现 `DataProvider`（例如时间/枚举/复杂结构）。
      - 测试与可观测性：覆盖本地/远端并发修改场景，使用 RequestID 串联日志与消息，补充失败重试、告警与审计日志，便于排查与补偿。
      - 常见修改点参考：`service/mtbl_user.go`（`DatasheetID`、`TableName`、`FirstCol`、`ColList`、`NewDO`）、回调入口实现位于 `ltblCallback`/`mtblCallback` 模块。

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
  - 控的方面更加倾向于用pgo来承载，同样使用corba交互操作
- 对于其他项目想要采用该项目的开发模式
  - bootCheck不要依赖orm，才能用于其他项目

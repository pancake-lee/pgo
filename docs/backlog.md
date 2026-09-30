# Backlog

> 全部技术需求池，按优先级排列。状态流转：`待规划` → `规划中` → `已规划` → `WIP` → `待用户验收` → `Done`；`暂缓` 表示已确认当前不执行。已完成事项见 [v0.0.10](archive/v0.0.10.md)。

## 任务总览

> 表头不能随意修改，即使表格清空了，也要保留表头，保留一个空表。

| 状态 | 分组 | 编号 | 任务 | 评估 |
| ---- | ---- | ---- | ---- | ---- |
| Done | 性能基线 | 23 | 多用户注册与登录的 HTTP 压测及观测闭环 | |
| 暂缓 | 代码生成 | 16 | genCURD 支持多主键表 | |
| 待规划 | 测试基础设施 | 22 | 外部客户端缺少可注入依赖与离线契约测试 | |
| 待规划 | 部署体验 | 24 | Docker Compose 网页组件统一入口 | |
| Done | 部署可靠性 | 25 | BootCheck 等待 RabbitMQ 和 Redis 就绪 | |
| Done | 代码生成 | 26 | genCURD 嵌套执行 make api 遗漏新生成 Proto | |
| Done | CLI 交互 | 27 | CI/CD 参数确认与批量跳过 | |

---

## 详细说明

### 23. 多用户注册与登录的 HTTP 压测及观测闭环

- **专题中枢**：[登录性能测试闭环](design/2026-09-30-01-login-performance-hub.md)
- **状态**：Done
- **背景**：`POST /user/token` 当前同时承担首次用户名注册和已有用户登录。现有 `TestUserService` 直接调用 Service，只覆盖单用户业务正确性，没有经过 HTTP、鉴权、请求中间件和真实并发，也缺少可重复的数据批次、数据库观测与实验记录。当前先模拟大量用户注册和登录，建立从业务行为、负载、指标到问题定位的第一条闭环；用户继续使用其他业务功能的场景待本任务完成后再按顺序规划。
- **分析**：HTTP 中间件已经提供请求量、错误率和延迟指标，诊断端口已经提供 Go 运行时、进程指标与 pprof；MySQL 用户名已有唯一索引。当前缺口是有状态场景驱动、独立清理、数据库查询耗时和连接池指标。首轮只记录实际现象，不预设瓶颈位于 Go、MySQL、连接池或网络。
- **方案**：增加面向本地实验的专用 Go 场景工具，通过 Swagger SDK 调用真实 `/user/token` 接口，创建带批次标识的用户并保存用户 ID、用户名和 token 清单；清理时使用批次清单和 SDK 提供的受鉴权用户删除接口，不直接写 SQL。场景工具负责注册、响应取值和清理等有状态流程，固定登录负载使用仓库外安装的 `vegeta v12.13.0`，不加入 `go.mod`。在现有 Prometheus 诊断端口补充通用的数据库查询耗时与连接池状态指标，实验期间同时采集 HTTP RED 指标、Go 运行时、进程资源和 pprof。基线记录保存到 `docs/eval/performance/`，只提交参数、摘要、结论和必要图表，不提交 token、真实配置、大体积原始结果或 profile。
- **第二阶段方案**：在 `pgo` 一级菜单增加可扩展的 `performance` 入口，首个场景为 `performance login`。现有 `user-load` 不再作为独立 tools 子项或公开命令，用户准备、正确性验证、目标生成和批次清理代码迁入登录性能场景，成为一次自动化运行的内部阶段。`performance login` 接收 API 地址、diagnostics/pprof 地址、用户数量、注册并发、RPS、预热与持续时间、请求超时和输出目录，一次执行完整闭环。负载继续调用固定版本的外部 Vegeta CLI，不把 Vegeta 加入 `go.mod`；代码调用处增加明确注释，记录未来可评估改用 Vegeta Go 包内嵌执行。正式负载运行期间由 Go 统一协调 metrics、CPU、heap 和 goroutine 采集，任一阶段失败时停止本次负载、保留已有诊断产物，并按批次清单尝试清理。输出目录同时保存原始数据和面向阅读的摘要，命令结束时逐项打印文件路径、内容与用途。
- **固定实验条件**：
  - 用户数据分为 100、1,000、10,000 三档；用户名使用随机批次 ID 前缀，清单写入被 Git 忽略的本地结果目录，重复运行不得碰触非本批次数据。
  - 注册阶段由场景工具控制并发，记录成功数、失败数和端到端耗时；登录阶段只使用已注册用户，分别以 10、50、100 RPS 持续 60 秒，正式采样前预热 10 秒。
  - 每轮固定代码提交、配置摘要、Go/MySQL/vegeta 版本、批次 ID、用户数、并发、RPS、持续时间、超时及开始结束时间；敏感配置仅记录是否就绪，不记录值。
  - 每轮结束后校验成功响应中的用户身份和 token 可用性，再清理该批次；失败或中断后可凭清单幂等重试清理。
- **任务列表**：
  - 建立场景工具的 `prepare`、`verify`、`cleanup` 入口；`prepare` 通过 `/user/token` 注册用户并生成本地批次清单，`verify` 重放登录并检查用户身份与 token，`cleanup` 通过现有 HTTP 删除接口精确删除批次用户。
  - 为场景工具增加离线 HTTP 假服务测试，覆盖批次命名、响应解析、并发错误汇总、清单恢复和幂等清理；保留现有 MySQL 集成测试作为业务层回归。
  - 增加数据库查询耗时直方图与连接池打开、使用中、空闲、等待次数和等待时长指标；标签保持低基数，不记录 SQL、用户名、用户 ID 或批次 ID。
  - 提供 `vegeta v12.13.0` 的安装与版本校验说明、登录目标生成入口，以及预热和 10/50/100 RPS 分档命令；目标和结果写入本地结果目录。
  - 以小档数据先验证 HTTP、鉴权、指标和清理闭环，再依次执行三档数据与阶梯负载；采集 CPU、heap、goroutine profile，并按同一模板记录吞吐、错误率、延迟分位数、资源和数据库指标。
  - 只在证据能够定位问题后新增修复子任务；修复前后使用相同数据批次规格与负载参数复测，并重新验证注册、登录及清理结果。
  - 移除根命令中的 `user-load` 注册，将 `cmd/pgo/tools/userload/` 的批次模型、HTTP 操作和测试迁入 `performance login` 模块；不保留 `prepare`、`verify`、`targets`、`cleanup` 四个公开子命令，相关能力只作为登录压测内部阶段使用。
  - 复用 diagnostics 的底层下载能力，但不从 `performance login` 嵌套调用 Cobra 命令；一级 `performance` 只注册场景，登录子命令负责本场景的参数与编排。
  - 增加 Vegeta 可执行文件和版本预检，通过受控子进程执行预热、正式负载和原始报告；调用处注释说明未来可替换为 Vegeta Go 包，但本阶段不增加该依赖。
  - 在正式负载窗口内协调指标采样和 CPU、heap、goroutine profile：CPU 持续采样，heap 与 goroutine 在负载期间取快照；确保命令退出前等待或终止所有子进程和 goroutine。
  - 设计单一输出目录结构，保存运行参数、批次清单、Vegeta 原始结果与报告、metrics 原始快照、三个 profile 和综合摘要；摘要解释吞吐、成功率、延迟分位数、HTTP、Go 运行时、进程、数据库和连接池指标的含义，并标记缺失或采集失败项。
  - 增加离线编排测试，使用假 HTTP 服务和可注入的 Vegeta 命令执行器覆盖成功流程、负载失败、采集失败、取消、清理和无遗留进程；同步 README 与基线文档，删除独立 `user-load` 命令说明，统一展示 `performance login` 自动化入口及其内部阶段。
- **验收**：
  - 用户数量、准备并发和 RPS 可通过参数调整；代表性真实运行可通过 HTTP 创建、验证和精确清理批次用户，失败批次可凭清单恢复清理，且不影响非本批次数据。
  - 无 token 和无效 token 的受保护请求被拒绝；注册返回的 token 能调用受保护接口；登录响应中的用户身份与请求目标一致。
  - 10、50、100 RPS 登录实验可由固定命令复现，结果包含实际吞吐、成功率、P50/P95/P99、CPU、内存、goroutine、数据库查询耗时与连接池状态。
  - CPU、heap、goroutine profile 均能在负载采样窗口获取；服务和负载命令以前台或限时方式执行，结束后无遗留进程。
  - 单次 `pgo performance login` 可根据参数完成完整闭环；结束输出列出全部产物及其意义，综合摘要无需手工拼接即可阅读本轮条件、结果和关键指标含义。
  - `performance` 可继续注册其他独立压测场景，登录场景的用户准备与结果解析不会成为其他场景的公共前提。
  - `GOTOOLCHAIN=local make test`、场景工具离线测试和现有 userService 集成测试通过；基线记录说明环境、步骤、观察、证据和阶段性结论，不包含敏感信息。
- **后续边界**：本任务只建立注册与登录闭环。完成后再从部门与职位、项目成员、角色权限等真实业务中选择一个场景，沿用同一套批次、负载和观测方法；当前不并行建立这些后续任务。
- **实施与验证**：第一阶段建立了用户批次、diagnostics 和数据库指标能力。第二阶段将批次逻辑迁入 `cmd/pgo/tools/performance/login/`，移除公开 `user-load` 命令，新增非交互式 `pgo performance login`，并在 `pgo` 顶层交互菜单增加“性能测试 → 用户登录压测”；两种入口调用同一 runner，交互参数使用现有缓存机制。程序统一编排固定版本 Vegeta CLI、metrics、CPU/heap/goroutine profile、摘要生成和失败清理；外部 Vegeta 调用处保留未来改用 Go 包的注释。控制流量关闭 HTTP keep-alive，避免旧服务未声明关闭连接时产生交替 EOF；Vegeta 被测流量保持默认连接行为。输出目录保存 11 类程序产物，文件名按阶段使用 `00–40` 编号并与 `docs/eval/performance/user-login/` 样例一致，shell `>` 可另存 CLI 自身输出；终端先单独打印输出目录，随后只打印文件名与用途，避免重复父目录。离线假服务覆盖完整编排与清理，定向竞态测试、全仓测试、`go vet`、构建和 CLI help 检查通过。
- **真实验证**：2026-09-30 使用 API `192.168.3.18:20000`、pprof `192.168.3.18:20002`、100 用户、准备并发 10、10 RPS、预热 10 秒和正式负载 60 秒完成闭环。600 个请求全部返回 200，吞吐 10.01 RPS，P50/P95/P99 为 13.846/21.343/32.698 ms；服务端登录平均 1.038 ms，数据库查询平均 731 µs，连接等待 0，goroutine 为 28→28。CPU、heap、goroutine profile、负载前后 metrics、Vegeta 原始结果与精简摘要均生成，批次用户清理完成，无 Vegeta 或 performance 进程遗留。
- **后续实验**：1,000/10,000 用户和 50/100 RPS 属于使用本工具扩展性能基线的后续实验档位，不阻塞自动化能力关单；每档使用独立输出目录并保留相同参数记录。

### 16. genCURD 支持多主键表

- **状态**：暂缓
- **背景**：`newTable` 检测到多个主键后将 `PriCol` 置空；DAO、Proto、Service 模板据此移除主键相关的 Update/Delete 逻辑。
- **方案**：将生成器的单一 `PriCol` 模型扩展为有序主键列集合；Update/Delete 请求、服务转换和 DAO 条件都包含全部主键，并只以完整主键组合作为定位条件。保留无主键表的只读生成行为，并以 SQLite 和 MySQL 的联合主键样例回归。
- **任务列表**：
  - 重构表元数据和模板替换逻辑，使其接受主键列集合。
  - 更新 Proto、Service、DAO 生成结果和零值/类型校验。
  - 建立单主键、联合主键、无主键三组 fixture，并验证生成代码可编译及 SQL 条件完整。
- **验收**：联合主键表生成完整 Update/Delete；SQL 条件包含且仅包含全部主键；单主键和无主键行为不回归。
- 决策：不想搞这个了，要么给表加上id列，要么自己写复合主键的CRUD，没必要折腾生成。

### 22. 外部客户端缺少可注入依赖与离线契约测试

- **状态**：待规划
- **背景**：任务 18 采用测试标签先恢复默认离线门禁。`papitable`、`predis`、`pweixin` 等客户端仍主要通过全局配置和真实连接初始化，无法充分覆盖请求构造、错误转换与响应解析等离线行为。
- **严重程度**：中。
- **关联**：任务 18 的后续演进；暂不纳入当前隔离改造范围。

### 24. Docker Compose 网页组件统一入口

- **状态**：待规划
- **背景**：`deploy/docker/docker-compose.yaml` 中的 RabbitMQ、Swagger UI、Prometheus、Grafana、cAdvisor 等组件分别提供网页入口，目前需要记忆并手动访问各自端口。后续增加一个统一入口页面，集中展示这些组件并提供跳转，降低本地开发和运维时查找入口的成本。
- **期望**：启动 Docker Compose 环境后，可从一个固定地址进入导航页，并从中跳转到各个已配置的网页组件；组件增删或端口调整时，入口信息应便于同步维护。
- **待规划项**：确定入口页面的承载方式、组件清单与展示信息、访问地址生成规则，以及不可用组件的呈现方式。

### 25. BootCheck 等待 RabbitMQ 和 Redis 就绪

- **状态**：Done
- **背景**：首次 Docker Compose 部署时，RabbitMQ 容器已进入运行状态，但 AMQP 端口尚未监听；Rocky 9 仅依赖容器启动顺序，BootCheck 立即连接后收到 `connection refused`，并可能在 RabbitMQ 就绪前耗尽容器重启次数。
- **方案**：为 MySQL、Redis 和 RabbitMQ 增加容器健康检查，并让 Rocky 9 等待三个依赖全部健康后再启动；同时让 BootCheck 对 RabbitMQ 和 Redis 进行限时重试，覆盖 Compose 之外的启动时序，MySQL 沿用已有重试。容器启动脚本显式启用 BootCheck 控制台日志，使每次失败和最终错误可由 Docker 日志查看。
- **任务列表**：
  - 增加 MySQL、Redis 和 RabbitMQ healthcheck，并将 Rocky 9 的三个依赖全部改为 `service_healthy`。
  - 为 RabbitMQ 和 Redis 检查增加固定次数、固定间隔的重试，最终错误保留依赖名与最后一次失败原因。
  - 验证重试成功、重试耗尽、Compose 配置和全仓回归。
- **验收**：首次启动时 Rocky 9 不会在 RabbitMQ 就绪前执行 BootCheck；RabbitMQ 或 Redis 短暂未就绪时 BootCheck 可在限定时间内恢复；持续不可用时容器日志包含具体依赖和最终连接错误。
- **实施与验证**：MySQL、Redis 和 RabbitMQ 分别使用容器内的 `mysqladmin ping`、`redis-cli ping` 和 `rabbitmq-diagnostics -q ping` 执行健康检查，Rocky 9 等待三者全部健康；BootCheck 对 RabbitMQ 和 Redis 最多尝试 12 次、间隔 5 秒，最终错误保留最后一次失败原因；启动脚本使用 `bootCheck -l` 输出容器日志。重试单测、启动脚本语法、全仓测试、`go vet` 和全量构建通过；当前环境没有 Docker CLI，Compose 解析与真实启动留待部署环境验收。
- **（用户）验收操作**：使用 PGO CD 更新部署文件和 `bootCheck`，重新执行 `docker compose up -d`，观察 `docker compose ps` 与 `docker compose logs rocky9`。
- **预期结果**：MySQL、Redis 和 RabbitMQ 全部进入 healthy，随后 Rocky 9 启动；短暂未就绪时日志显示重试并最终成功，服务进程正常运行。
- **最小回传**：成功时回复“25 已通过”；仍失败时回传 `rocky9` 从首次依赖检查到最终错误的日志片段。
- **AI 自动验证**：`GOTOOLCHAIN=local make test`、`GOTOOLCHAIN=local make build` 和 `sh -n deploy/docker/config/startup.sh` 均通过。
- **关单方式**：用户回复确认后，直接将任务 25 更新为 `Done` 并注明确认日期，不追加核验。
- **用户确认**：2026-09-29，部署环境验收通过，任务关单。

### 26. genCURD 嵌套执行 make api 遗漏新生成 Proto

- **状态**：Done
- **背景**：从 `make curd` 启动 `pgo genCURD` 时，生成器内部调用 `make api` 报告手写 Proto 导入的 `z_userService.gen.proto` 不存在；失败回滚后单独执行 `make api` 可成功。
- **分析**：真实 MySQL 回归确认工作目录正确，且子 make 收到了调用时实际存在的 Proto 清单。根因是任务 14 引入 `pgo.tables` 后，当前项目没有将 user、task、school 表归属迁移到手写 Proto；生成器因此将对应的 `z_*Service.gen.proto` 当作过期文件删除。随后的回滚恢复了旧文件，造成手工 `make api` 可成功的表象。
- **方案**：保留 `make api` 作为唯一 API 生成入口，`genCURD` 显式传入生成和清理完成后的最新 Proto 清单，并固定子进程项目根目录。按现有 Proto 表归属契约为 user、task、school 补齐 `pgo.tables` 声明，修正 option 生成包路径和 default 映射插入位置，并在 README 补充面向使用者的操作说明。
- **任务列表**：
  - 收敛 API 生成调用的项目根目录和 Proto 文件清单，保证清单反映本轮生成后的文件状态，且顺序稳定。
  - 调整 `make api` 的调用契约，允许 `genCURD` 显式传入 Proto 清单，同时保留用户直接执行 `make api` 时的自动发现行为。
  - 增加不依赖真实数据库和 protoc 的回归测试，覆盖新生成 Proto、清理过期 Proto、工作目录和子 make 失败信息。
  - 运行定向测试、全仓测试和构建，确认生成失败时的回滚行为不回归。
- **验收**：
  - 从 `make curd` 启动时，同一轮新建的 `z_*Service.gen.proto` 全部进入 protoc 输入，不再需要手工补跑 `make api`。
  - 已删除的过期生成 Proto 不会残留在 protoc 输入中，文件清单在相同输入下顺序一致。
  - 从项目外部通过 `workDir` 执行 `pgo genCURD` 与在项目根目录执行的结果一致；单独执行 `make api` 仍可自动发现所有 Proto。
  - API 生成失败时返回可诊断错误，并恢复本轮修改前的生成文件；`GOTOOLCHAIN=local make test` 和 `GOTOOLCHAIN=local make build` 通过。
- **实施与验证**：`genCURD` 实时收集并排序 Proto 清单，通过命令行变量传给固定工作目录的子 make；项目 Proto 已持久化 user、task、school 表归属，`pgo.tables` 扩展生成到独立 `api/pgo` 包；`abandonCode.proto` 增加仅用于模板展示的表归属 option，生成时会移除该示例，README 和 `abandonCodeService` 说明已增加互相引导。使用真实 MySQL DSN 执行新 CLI 的完整 `make curd` 链路成功，无需补跑 `make api`，且模板映射未泄漏到生成 Proto；定向测试、`GOTOOLCHAIN=local make test` 和 `GOTOOLCHAIN=local make build` 均通过。

### 27. CI/CD 参数确认与批量跳过

- **状态**：Done
- **背景**：PGO 的 CI 和 CD 交互式子命令会为多个参数依次提问。这些参数已有代码默认值或上次执行的缓存值，但用户在无需修改时仍必须连续回车，增加了重复操作。
- **分析**：`pkg/pclient` 已封装参数定义、默认值与缓存值合并逻辑，CI 的 Init Project 已使用该入口；CI 的 Make 变量和 CD 的 SSH 参数仍各自组织交互。本需求只改变交互式参数确认流程，不改变 Cobra 非交互命令的参数行为。
- **方案**：扩展现有的批量参数读取能力，先按“缓存值优先，否则代码默认值”计算全部当前有效值，再统一展示参数摘要并询问是否直接使用。该问题默认为是，用户直接回车或输入 `y` 时跳过后续所有参数提问，输入 `n` 时才按原有顺序逐个编辑；编辑时继续以当前有效值作为默认值并仅在值变化时更新缓存。摘要中普通参数显示实际值，SSH 密码等敏感参数只显示“已设置/未设置”。CI Make、CI Init Project 和 CD Deploy 共用该流程，保留各子命令现有参数来源、顺序和缓存键。
- **任务列表**：
  - 在现有参数交互封装中增加当前有效值汇总、敏感值脱敏和“直接使用/逐个修改”分支，不引入新的交互依赖。
  - 让 CI Make、CI Init Project 和 CD Deploy 复用同一套批量确认逻辑，清理重复的默认值与缓存读取代码。
  - 增加可注入的交互测试，覆盖回车跳过、`y` 跳过、`n` 后逐个编辑、默认值/缓存值优先级、缓存更新与密码脱敏。
- **验收**：
  - 进入 CI Make、CI Init Project 或 CD Deploy 时，在逐个提问前能看到全部待输入参数的当前有效值，敏感值不以明文出现。
  - 在确认问题上直接回车或输入 `y` 后，立即沿用当前值继续执行，不再逐个询问参数。
  - 输入 `n` 后，按原有顺序逐个询问；单个参数直接回车仍使用已展示的当前值，新值在后续执行中生效并按原有规则缓存。
  - Cobra 非交互调用、参数名、缓存键和 CI/CD 后续执行行为保持不变；定向测试、`GOTOOLCHAIN=local make test` 和 `GOTOOLCHAIN=local make build` 通过。
- **实施与验证**：`pkg/pclient` 先合并默认值与缓存值，统一输出参数摘要并提供默认为是的批量确认；选择修改时仍按原顺序逐项输入，只缓存相对当前有效值的变化。Make 变量、Init Project 与 CD SSH 参数共用该流程，SSH 密码摘要仅显示“set/not set”。离线测试覆盖回车与 `y` 跳过、`n` 后编辑、缓存优先级、变更缓存和密码脱敏；`go test ./pkg/pclient ./cmd/pgo/devops`、`make test` 和 `make build` 均通过。

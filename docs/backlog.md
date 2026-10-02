# Backlog

> 全部技术需求池，按优先级排列。状态流转：`待规划` → `规划中` → `已规划` → `WIP` → `待用户验收` → `Done`；`暂缓` 表示已确认当前不执行。已完成事项见 [v0.0.10](archive/v0.0.10.md)。

## 任务总览

> 表头不能随意修改，即使表格清空了，也要保留表头，保留一个空表。

| 状态 | 分组 | 编号 | 任务 | 评估 |
| ---- | ---- | ---- | ---- | ---- |
| Done | 性能基线 | 23 | 多用户注册与登录的 HTTP 负载闭环 | |
| 暂缓 | 代码生成 | 16 | genCURD 支持多主键表 | |
| 待规划 | 测试基础设施 | 22 | 外部客户端缺少可注入依赖与离线契约测试 | |
| Done | 部署体验 | 24 | Docker Compose 网页组件统一入口 | |
| Done | 部署可靠性 | 25 | BootCheck 等待 RabbitMQ 和 Redis 就绪 | |
| Done | 代码生成 | 26 | genCURD 嵌套执行 make api 遗漏新生成 Proto | |
| Done | CLI 交互 | 27 | CI/CD 参数确认与批量跳过 | |
| 待用户验收 | 可观测性 | 28 | Alloy、Pyroscope 与受控运行时诊断 | |
| Done | 性能基线 | 29 | 登录场景逐级加压与负载结果记录 | |
| Done | 性能基线 | 30 | 性能框架与登录场景解耦 | |
| Done | 性能基线 | 32 | 性能场景公共参数收口到 core | |

---

## 详细说明

### 23. 多用户注册与登录的 HTTP 负载闭环

- **专题中枢**：[登录性能测试闭环](design/2026-09-30-01-login-performance-hub.md)
- **状态**：Done
- **背景**：`POST /user/token` 同时承担首次注册和已有用户登录，原有测试只直接调用 Service，未覆盖真实 HTTP、鉴权与并发负载。
- **方案**：通过真实 HTTP 创建带批次标识的测试用户，验证身份和 token 后生成 Vegeta targets；使用固定版本 Vegeta 预热并制造登录负载，保存批次清单、原始结果和文本报告，最后通过受鉴权接口精确清理本批次用户。服务指标与 profiling 由 Grafana、Pyroscope 独立承担。
- **验收**：
  - 登录场景能够创建、验证和精确清理测试用户，失败或中断时仍尝试清理已创建数据。
  - Vegeta 负载经过真实 HTTP 和鉴权链路，保存请求数、吞吐、成功率与 P50/P95/P99。
  - 测试用户数据和 token 只写入被 Git 忽略的本地结果目录，不影响非本批次数据。
  - 场景离线测试、全仓测试、静态检查和构建通过，结束后无 Vegeta 或 performance 进程遗留。
- **实施与验证**：登录批次准备、身份验证、targets 生成、Vegeta 调用、结果保存与失败清理已集成到 `pgo performance login`。真实环境已完成 100 用户、10 RPS、60 秒验证，600 个请求全部成功，批次用户清理完成。后续固定阶梯与纯负载侧汇总由任务 29 维护。

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

- **状态**：WIP
- **背景**：任务 18 采用测试标签先恢复默认离线门禁。`papitable`、`predis`、`pweixin` 等客户端仍主要通过全局配置和真实连接初始化，无法充分覆盖请求构造、错误转换与响应解析等离线行为。
- **严重程度**：中。
- **关联**：任务 18 的后续演进；暂不纳入当前隔离改造范围。

### 24. Docker Compose 网页组件统一入口

- **状态**：Done
- **背景**：`deploy/docker/docker-compose.yaml` 中的 RabbitMQ、Swagger UI、Prometheus、Grafana、cAdvisor 等组件分别提供网页入口，目前需要记忆并手动访问各自端口。后续增加一个统一入口页面，集中展示这些组件并提供跳转，降低本地开发和运维时查找入口的成本。
- **期望**：启动 Docker Compose 环境后，可从一个固定地址进入导航页，并从中跳转到各个已配置的网页组件；组件增删或端口调整时，入口信息应便于同步维护。
- **方案**：在 Docker Compose 中增加独立的轻量 Nginx 导航容器，以只读方式挂载仓库内的静态页面，通过固定端口对外提供入口。导航页集中展示后端 API、pprof、RabbitMQ、Swagger UI、Prometheus、Grafana 和 cAdvisor，根据当前页面的主机名、各组件对外端口与可选路径生成 HTTP 跳转地址，适配本机和远程部署。页面在浏览器中执行轻量可达性探测，区分检测中、可访问和未响应；组件名称、说明、端口和路径集中在同一页面数据区维护。部署文件清单同步包含导航页目录。
- **任务列表**：
  - 增加 Nginx 导航服务和固定宿主机端口，挂载静态页面并加入现有 Compose 网络。
  - 实现组件卡片、动态主机地址、新窗口跳转和访问状态呈现。
  - 同步 CD 部署文件清单与 README 入口说明，执行 Compose、静态页面和全仓回归验证。
- **验收**：
  - 启动 Compose 后可从固定地址打开导航页，并跳转到后端 API、pprof、RabbitMQ、Swagger UI、Prometheus、Grafana 和 cAdvisor。
  - 通过 localhost 或远程主机名访问时，组件链接使用相同主机名，不依赖写死 IP。
  - 不可达组件仍保留入口并显示未响应状态；新增或调整组件时可在单一数据区同步导航信息。
  - Compose 配置、CD 部署映射、页面结构检查、`make test` 和 `make build` 通过。
- **实施与验证**：Compose 新增 `pgo-portal` Nginx 容器，以只读方式挂载 `deploy/docker/portal/` 并发布到宿主机 `20080` 端口。导航页的单一组件数据区包含后端 API、pprof、RabbitMQ、Swagger UI、Prometheus、Grafana 和 cAdvisor，根据当前主机名生成 HTTP 链接，支持 pprof 的 `/debug/pprof/` 子路径，并通过限时浏览器请求显示可访问或未响应。userService 新增无需鉴权的 `GET /`，返回 `hello, this is userService`，使后端 API 卡片存在明确的根地址响应。CD 文件映射和 README 入口说明已同步。回归测试锁定 Compose 入口端口、组件端口、pprof 路径、部署映射和根处理器响应；页面 JavaScript 语法检查、定向测试、`make test` 与 `make build` 均通过。
- **（用户）验收操作**：在部署主机更新 Compose 和 `portal/` 目录，执行 `docker compose up -d portal`，然后打开 `http://<部署主机>:20080`并任选一张显示“可访问”的组件卡片点击。
- **预期结果**：导航页显示七个入口及其状态，点击后在新窗口打开同一部署主机上的对应地址。pprof 未启用时该入口可显示未响应。
- **最小回传**：回复“24 已通过”；若失败，回传未打开的页面地址与 `docker compose ps portal` 输出。
- **AI 自动验证**：页面 JavaScript 语法检查、部署配置回归测试、`make test` 和 `make build` 均通过；本轮启动的测试与构建进程已全部退出。
- **关单方式**：用户回复确认后，同一轮将任务改为 `Done` 并注明确认日期，不追加核验。
- **用户确认**：2026-09-30，导航页其他验收项已通过；根处理器补充并完成自动验证后关单。

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

### 29. 登录场景逐级加压与负载结果记录

- **专题中枢**：[登录性能测试闭环](design/2026-09-30-01-login-performance-hub.md)
- **状态**：Done
- **背景**：`pgo performance login` 已完成纯负载职责收口，但导航页 `services.json` 的下载、校验和 URL 组合仍位于 performance 包，并且解析结果只暴露 API URL。这段能力描述的是整个 Compose 导航入口，不属于单一性能场景。
- **分析**：导航页清单是 CLI 公共基础信息。公共解析结果应与当前清单中的固定组件保持一致，集中生成 API、diagnostics、RabbitMQ、Swagger、Prometheus、Grafana、Pyroscope、Alloy 和 cAdvisor 的访问 URL；performance 只是其中一个调用方，只读取 API URL。组件 URL 采用导航页相同规则，以导航页协议和主机名组合端口，并优先使用 `webPath`，否则使用 `path`。
- **方案**：将服务清单模型、导航页 URL 校验、HTTP 获取、严格 JSON 解析、字段校验和组件 URL 组合整体迁移到 `cmd/pgo/common`。公共结果使用已选择的固定字段结构，对外提供当前九个组件的 URL。解析时要求九个稳定 ID 全部存在，拒绝重复 ID、非法端口、非法路径、未知字段和尾随 JSON；清单仍可包含展示名称与说明，但公共调用方只依赖生成后的 URL。performance 删除本地 services 实现与测试，通过公共入口取得结果并只消费 API URL，其负载、结果与观测职责边界不变。
- **任务列表**：
  - 在 `cmd/pgo/common` 建立导航服务清单解析工具和固定字段结果，覆盖当前 `services.json` 的九个组件。
  - 使用与导航页一致的 `webPath` 优先规则生成组件 URL，并覆盖带路径的导航页地址、IPv4、IPv6、HTTP 与 HTTPS。
  - 将清单下载状态码、响应体限制、严格 JSON 解析、重复 ID、完整组件、端口和路径校验迁入公共包。
  - 删除 `cmd/pgo/performance/services.go` 及其包内测试，调整 performance 只调用公共解析入口并读取 API URL。
  - 更新 README 与专题中枢中的公共服务发现说明，避免把 `services.json` 描述成 performance 专属协议。
  - 运行公共包和 performance 定向竞态测试、部署清单一致性测试、全仓测试、静态检查和构建。
- **验收**：
  - 公共解析结果对外提供 API、diagnostics、RabbitMQ、Swagger、Prometheus、Grafana、Pyroscope、Alloy 和 cAdvisor URL，结果与导航页实际链接一致。
  - diagnostics URL 使用 `webPath` 指向 heap 页面，其余组件在没有 `webPath` 时使用 `path`。
  - 缺少任一固定组件、ID 重复、端口或路径非法、未知字段、尾随 JSON 及非成功 HTTP 响应均返回可定位错误。
  - performance 包不再维护 services 清单模型、解析或 URL 拼装，只读取公共结果中的 API URL，负载行为和产物不回归。
  - 定向竞态测试、部署清单一致性测试、`make test`、`go vet ./...`、`make build` 与 CLI help 检查通过，测试结束后无进程遗留。
- **实施与验证**：导航页清单解析已迁入 `cmd/pgo/common`，公共固定字段覆盖九个组件 URL，并与页面统一采用 `webPath` 优先规则。performance 本地 services 实现与解析测试已删除，只读取公共结果的 API URL；单一 `Prepare` 能力统一以 `preparer` 命名，本仓库 Harness 已记录窄接口按能力命名规则。压测配置已删除无意义的登录包装层，API URL 归入通用负载配置，固定时长、超时和 Vegeta 路径直接使用内部常量，登录专属产物名归入准备器；外部命令执行复用 `pkg/putil` 的 context 与流式输出入口。压测执行器在构造时注入 Kratos Logger，后续调用只传业务参数；Vegeta 标准错误由进程适配器捕获并包装为错误。公共包、performance、登录场景与部署清单定向竞态测试、`make test`（含 `go vet ./...`）、`make build` 和 CLI help 检查通过，无本轮测试进程遗留。

### 30. 性能框架与登录场景解耦

- **专题中枢**：[登录性能测试闭环](design/2026-09-30-01-login-performance-hub.md)
- **状态**：Done
- **背景**：`cmd/pgo/performance` 同时承担 Vegeta 单档负载、自动升压、登录场景准备、Cobra 命令和交互入口。登录专属代码散落在通用包中，使单档负载与自动升压难以被其他场景复用。
- **分析**：Go 包不允许 `performance` 与其 `login` 子包相互导入。采用全局场景注册时，依赖方向固定为 login 依赖 performance；CLI 组合根只负责引入场景包以触发注册，performance 不反向依赖 login。注册表只在程序初始化期写入，运行期仅读，避免将它扩大为通用的动态插件系统。
- **方案**：将 performance 收口为通用性能测试框架，仅保留场景注册、单档 Vegeta 负载、通用自动升压与负载结果记录。场景通过紧凑的准备契约向单档执行器提供 targets 和清理动作，自动升压通过单档执行回调复用同一条执行链。login 包负责登录命令、交互入口、用户准备与清理、默认阶梯选择及登录汇总文案，并在初始化期将场景注册到 performance。
- **任务列表**：
  - 定义初始化期使用的场景注册契约，由 performance 统一生成 Cobra 根命令和交互菜单，并对重复或无效注册立即失败。
  - 将登录命令、交互入口与负载准备迁入 `cmd/pgo/performance/login/command.go`，登录包内继续复用现有用户批次能力。
  - 将 `automation.go` 改为与场景无关的固定阶梯执行和结果记录，每个档位回调 performance 的单档压测，不在通用包中保留登录命名或登录文案。
  - 将默认 RPS 阶梯、单档/自动模式选择和登录汇总内容收回 login 包，通过框架契约调用通用自动升压。
  - 调整 CLI 组合根对 login 场景的初始化引入，同步迁移测试到各自责任包，覆盖注册、单档复用、阶梯停止、失败清理和既有 CLI 行为。
  - 运行 performance/login 定向竞态测试、全仓测试、静态检查和构建，并检查无测试或 Vegeta 进程遗留。
- **验收**：
  - `cmd/pgo/performance` 不再包含登录命令、登录交互、用户准备或登录专属的自动化文案，`cmd/pgo/performance/login` 是登录场景的唯一实现位置。
  - 单档压测只负责一次场景准备、Vegeta 预热、正式负载、报告生成与清理；自动升压仅组织档位并通过该单档入口执行。
  - performance 与 login 之间无循环依赖，注册表只在初始化阶段变更，重复场景标识不会静默覆盖。
  - `pgo performance login <portal-url> [--rps]` 与现有交互菜单行为保持不变，单档及自动阶梯的产物、失败停止和清理行为不回归。
  - 定向竞态测试、`make test`、`go vet ./...`、`make build` 和 CLI help 检查通过，验证结束后无相关进程遗留。
- **实施与验证**：performance 已收口为初始化期场景注册、单档 Vegeta 执行和通用自动升压；注册表拒绝缺失字段和重复标识。login 通过初始化注册提供 Cobra 命令与交互入口，并在 `command.go` 内统一管理导航服务发现、RPS 模式、默认阶梯、用户准备清理和登录汇总文案。自动升压通过单档回调复用同一执行链，通用包不再包含登录符号或文案；CLI 组合根仅匿名引入 login 以触发注册。场景注册、单档产物与清理、阶梯回调、执行失败和非 100% 成功率停止、登录准备清理与 CLI 契约均有回归测试。`go test -race ./cmd/pgo/performance/... ./cmd/pgo`、`make test`（含 `go vet ./...`）、`make build` 及 performance/login help 检查通过，结束后无 Go 测试、pgo performance 或 Vegeta 进程遗留。
- **可读性整理**：2026-10-02 将 performance/login 生产代码与测试统一限制为 80 字符行宽，拆分 `if` 初始化语句，并按校验、准备、执行、结果与清理阶段留白。函数声明在仅返回部分导致超长时保留同行参数；函数调用可容纳时完整单行，否则从左括号后换行，每个实参独占一行。定向竞态测试、`make test`（含 `go vet ./...`）与 `make build` 通过。

### 32. 性能场景公共参数收口到 core

- **专题中枢**：[登录性能测试闭环](design/2026-09-30-01-login-performance-hub.md)
- **状态**：Done
- **背景**：当前 `portal-url` 和 `rps` 由 login 命令读取与解析，`duration` 则在 core 内固定为 60 秒。这三项都是所有性能场景共用的运行参数，继续放在 login 会让后续场景重复实现 CLI、交互输入和校验逻辑。
- **分析**：公共参数的输入形式、默认值、校验和转换应只有一个事实来源。core 已经拥有单档负载与自动升压执行链，由它同时生成 Cobra 和交互入口，可以使场景只描述自身准备行为与自动升压档位。`portal-url` 解析后得到的 API URL、单档 RPS 与持续时间统一进入通用负载配置；自动模式只替换各档 RPS，每档共用同一持续时间。
- **方案**：由 core 提供统一的性能场景入口，集中定义 Cobra 参数、交互输入、缓存读取、服务发现、默认值和错误提示。公共入口读取 `portal-url`、`rps` 和 `duration`，其中 RPS 保留正整数单档与 `auto` 两种模式，duration 使用 Go 时长格式并保留 60 秒默认行为。场景向 core 提供名称、输出目录分段、准备器、自动档位和汇总文案；login 删除通用参数处理与执行编排，仅保留用户准备、验证、targets 生成及清理。
- **任务列表**：
  - 在 core 建立性能场景的公共入口和场景配置契约，统一生成 Cobra 命令与交互执行流程。
  - 将 `portal-url`、`rps` 和 `duration` 的读取、默认值、解析与校验迁入 core，并将解析结果写入通用单档负载配置和运行产物。
  - 调整单档与自动升压执行链使用指定 duration，自动模式每档保持同一时长。
  - 收缩 login 入口，通过场景契约提供登录专属信息与准备器，删除其中的公共参数解析、服务发现和负载编排。
  - 迁移测试责任：core 覆盖三项参数的 CLI、交互、非法输入、默认值与单档/自动传递，login 只覆盖场景契约和准备清理。
  - 同步 CLI help、性能基线说明和专题中枢，运行定向竞态测试、全仓测试、静态检查与构建。
- **验收**：
  - 新增性能场景时无需重复定义或解析 `portal-url`、`rps` 和 `duration`，Cobra 与交互模式共用同一套规则。
  - `pgo performance login <portal-url> --rps <positive|auto> --duration <duration>` 可用，未指定时保留现有自动阶梯和 60 秒单档时长。
  - 无效 portal URL、非正整数 RPS、非法或非正 duration 均在 core 返回可定位错误，不进入场景准备。
  - 单档与自动升压的预热、正式负载、失败停止、结果产物和清理行为不回归，运行记录包含实际 duration。
  - core 和 login 定向竞态测试、`make test`、`go vet ./...`、`make build` 及 CLI help 检查通过，验证后无相关进程遗留。
- **实施与验证**：core 已新增通用场景入口，统一生成 Cobra 命令和交互参数，并负责 portal 服务发现、RPS 单档/自动模式解析、duration 校验、输出目录和单档/自动执行编排。`Config` 已保存 duration，每个自动档位复用同一时长；固定预热策略保持不变。login 已收缩为场景元数据、自动阶梯、准备器与用户数据生命周期。公共 Portal 解析后续已支持无协议的主机与端口，默认补全 `http://`，并保留 HTTP/HTTPS 显式协议校验。定向竞态测试、全仓 `make test`（含 `go vet ./...`）、`make build` 和 CLI help 检查通过，默认参数为 `--rps auto --duration 60s`。

### 31. pclient 两层命令菜单

- **状态**：Done
- **背景**：performance 自建场景注册、Cobra 父命令和交互菜单，与 `pkg/pclient` 已有工具入口形成平行机制；现有一级交互菜单还混有可直接执行的功能。目标结构固定为一级分组、二级具体工具，Cobra 与交互模式共用同一份结构。
- **方案**：在 `pkg/pclient` 增加只负责双运行模式组合的 `CommandGroup` 与 `CommandEntry`。`cmd/pgo/main.go` 定义并注册 DevOps、Performance、Application 和 Tools 一级分组；各业务分组根目录登记自身拥有的二级工具；具体工具只暴露 Cobra 与交互入口并处理自身参数，不感知所在分组。diagnostics 原三级命令拆为 Tools 下的 health、metrics-url 和 profile 二级工具。为避免 performance 根包注册 login 时产生循环依赖，通用压测执行框架下沉到 `performance/core`，分组入口保留在 `performance/menu.go`。
- **验收**：
  - `pgo` 层只定义四个一级分组的名称和展示文案，不直接挂载可执行工具。
  - devops、performance、application 和 tools 各自登记二级工具，具体工具不依赖菜单层级。
  - performance 删除 `Scenario`、`RegisterScenario`、`NewCommand` 和 `RunInteractive`。
  - 所有 Cobra 与交互入口均为固定两层结构，并复用相同的二级工具列表。
- **实施与验证**：已完成 pclient 分组与工具入口抽象以及全部 pgo 入口迁移。Performance 分组入口位于根目录，通用压测框架位于 `performance/core`；Course Swap 迁入 `application/courseSwap`，由 Application 根包登记；开发、诊断和交互测试工具归入 Tools；无调用的 `cmd/pgo/common` 旧入口与参数副本已删除。两层 Cobra 树测试覆盖全部工具路径；`go test -race ./cmd/pgo/... ./pkg/pclient`、`make test`（含 `go vet ./...`）、`make build` 以及四个分组的 CLI help 检查全部通过。

### 28. Alloy、Pyroscope 与受控运行时诊断

- **状态**：待用户验收
- **背景**：部署侧需要以持续、独立于压测命令的方式保存服务日志、Prometheus 指标和 CPU/heap profile，同时限制 goroutine、block、mutex 与 runtime trace 的开放时间。
- **分析**：Grafana、Prometheus、Loki、Alloy 和 Pyroscope 是服务侧观测链路。它们应持续工作并按时间范围关联分析，不依赖 performance 命令触发或导出数据。2026-10-02 真实部署发现 Pyroscope 以非 root 用户运行时无法在宿主机 bind mount 中创建 `segments`，导致 profile 写入返回 500；当前 Compose 已对同类 Loki 数据目录显式使用 root 用户，Pyroscope 配置遗漏了相同处理。
- **方案**：保留 Prometheus 指标链路；在 Docker Compose 中由 Alloy 替代 Promtail，将文件日志写入 Loki，并持续拉取应用 CPU、heap profile 写入 Pyroscope；Grafana 预置对应数据源。应用诊断服务长期提供健康检查、metrics、CPU 和 heap，goroutine、block、mutex 通过最长 60 秒的受控接口按需开放，runtime trace 通过最长 10 秒的独立接口采集。Pyroscope 与 Loki 一致以 root 用户运行，使其能够初始化宿主机 bind mount 下的持久化子目录；不使用放宽全目录权限的方式绕过问题。该任务不向 performance 包提供采集、导出或报告职责。
- **任务列表**：
  - 验证 Alloy、Loki、Prometheus、Pyroscope 与 Grafana 的部署、持久化、数据源和导航入口。
  - 验证 CPU、heap 持续采集以及 goroutine、block、mutex、runtime trace 的限时开放和自动恢复。
  - 验证 Promtail 已移除，Alloy 可从持久化位置续读日志，Loki 存储目录权限正确。
  - 修正 Pyroscope 容器用户，验证空数据目录首次启动时可创建持久化子目录并接收 profile。
  - 运行诊断处理器测试、Compose 配置检查、全仓测试、竞态测试和构建。
- **验收**：
  - Grafana 可连接 Prometheus、Loki 和 Pyroscope，服务重启后历史日志和 profile 数据仍保留。
  - Alloy 持续采集 CPU、heap，Pyroscope 可按服务和时间范围查询，采集不依赖 performance 命令。
  - goroutine、block、mutex 未开启时不可读取，合法租约到期、取消或服务停止后恢复关闭；runtime trace 最长 10 秒且并发冲突被明确拒绝。
  - Promtail 已移除，Alloy 重启后续读日志，Grafana 可通过 Loki 查询带原有时间戳与应用标签的新日志。
  - Pyroscope 不再出现创建 `/var/lib/pyroscope/segments` 权限不足，Alloy 推送 profile 不再因此返回 500。
  - 自动测试、竞态测试、Compose 配置检查和构建通过，真实部署验收后无本轮启动的诊断进程遗留。
- **实施与验证**：诊断服务、Compose、Alloy 日志与 profiling、Grafana 数据源和导航入口已完成代码侧实现与自动验证；Loki bind mount 权限问题已修正。2026-10-02 宿主机验收暴露 Pyroscope bind mount 权限遗漏，Compose 已将 Pyroscope 调整为与 Loki 相同的 root 用户，并增加部署配置回归断言。定向测试、`make test`（含 `go vet ./...`）和 `make build` 通过；当前环境缺少 Docker CLI，未执行 Compose 解析和真实容器验证。
- **（用户）验收操作**：在宿主机的 `deploy/docker` 目录执行 `docker compose up -d --force-recreate pyroscope alloy`，等待 Alloy 完成下一轮采集后，在 Grafana Explore 中查询同一时间窗的 CPU 或 heap profile。
- **预期结果**：Pyroscope 日志不再出现创建 `/var/lib/pyroscope/segments` 权限不足，Grafana 可查询到新的 CPU 或 heap profile。
- **最小回传**：成功时回复“28 已通过”；失败时回传 `docker compose logs --tail=100 pyroscope alloy`。
- **AI 自动验证**：部署配置定向测试、全仓测试、静态检查和构建均已通过；未启动宿主机部署服务。
- **关单方式**：用户回复确认后，同一轮将任务 28 更新为 `Done` 并注明确认日期，不追加核验。

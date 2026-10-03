# 测试策略

> 测试策略、测试分层、自动化方案、发布前检查清单。

## 当前实践

- 验证频率遵循 [分阶段验证节奏](handbook/work-modes.md#验证节奏所有模式)，不在每轮代码生成后重复执行完整测试。
- 编写集成测试验证 Service 层逻辑
- 使用 `defer` + 清理函数移除测试数据
- **禁止**在测试中修改表结构，差异应正常报错以提醒升级注意
- 多用户注册与登录使用真实 HTTP 场景工具和固定 Vegeta 负载，实验方法见[多用户注册与登录基线](eval/user-login-baseline.md)

## 权限合并性能练习

设计见[权限场景](design/2026-10-03-01-user-permissions-performance.md)，交接与验收见[任务 47](backlog.md#47-用户权限聚合性能实验)。本场景已接入 CLI 与 Performance 交互菜单。

### 启用服务

编译客户端和用户服务，产物位于 `bin/`：

```shell
go build -o ./bin/pgo ./cmd/pgo
go build -o ./bin/userService ./internal/userService
```

将新 `userService` 放到当前部署使用的服务路径，在该服务实际读取的配置文件中设置以下顶层开关，然后按现有方式重启服务：

```yaml
PermissionExercise: true
```

仓库模板默认关闭练习；开启后普通请求仍使用 map，只有带练习请求头的请求会使用 list。客户端在准备大批数据前检查模式回传，旧服务或未开启练习会立即报错并清理本批次用户。

本地前台运行示例为 `./bin/userService -c .local/my-config.yaml -l`，需要现有数据库、消息队列等配置已就绪，Ctrl+C 停止。已经通过 Docker/PM2 运行服务时沿用部署启动方式，不同时启动第二个同端口服务。

### 跑低效版本，再跑修复对照

下列 `http://127.0.0.1:20080` 是导航页地址；远程部署时替换为当前 login 压测使用的同一个导航地址。

```shell
./bin/pgo performance permissions http://127.0.0.1:20080 --roles 10 --permissions 500 --aggregation list --rps 10 --duration 120s
./bin/pgo performance permissions http://127.0.0.1:20080 --roles 50 --permissions 500 --aggregation list --rps 10 --duration 120s
./bin/pgo performance permissions http://127.0.0.1:20080 --roles 50 --permissions 500 --aggregation map --rps 10 --duration 120s
```

每次自动创建唯一批次用户、目标项目和角色权限，另建一个项目检查隔离；核对结果后预热 30 秒，再执行正式负载，最终删除批次数据。角色间重复权限保持同样路径，最终均返回 500 个动作。list 与 map 运行会重新创建逻辑内容相同的数据，数据库 ID 随批次变化。

角色数支持 1 至 50，每角色权限数支持 1 至 2000。10 角色搭配 100、500、2000 个权限可练习唯一动作规模的影响。正式比较同条件重复三次。

需要升压时保持其他参数一致，仅将 `--rps` 改为 25、50、100 等单档值。公共 `--rps auto` 当前从 200 RPS 起步，先完成低速率实验再使用；本场景默认 10 RPS。

### 对齐图表与业务热点

结果默认写入 `.local/performance/permissions/<UTC 时间戳>/`，交互模式可自定义目录。自动阶梯每档独立保存。关键文件为：

- `01-permissions.json`：角色数、权限数、list/map、数据 ID 和清理状态，包含本批次 token，文件权限为 0600。
- `02-permissions-targets.jsonl`：带鉴权和算法选择的 GET 请求目标。
- `11-vegeta-report.txt`：完成吞吐、成功率和 P50/P95/P99。
- `12-load-window.json`：正式负载的 UTC 起止窗口，排除数据准备与预热；结束时间包含负载进程等待在途请求完成的尾段。

从 `12-load-window.json` 选择同一时间范围：

- **Grafana，PGO Application**：选择服务实例，请求 operation 选 `/api.User/GetUserPermissions`；看 Request rate、Error rate、Request p95 by operation、Process CPU，再看 Database operation rate 和 Connection wait rate。数据库操作选择 `query`，数据库指标无接口维度，比较时停止其他业务负载。
- **Pyroscope，CPU**：选择 `service_name="pgo-app"` 和对应实例（以实际标签为准），搜索 `mergePermissionsByList`，展开调用链中的业务合并和字符串比较。map 对照中看这个热点是否消失，同时核对绝对 CPU 与 P99，避免只比较栈占比。
- **Pyroscope，heap**：`alloc_space` 辅助观察列表分配与 ORM 扫描开销；它不是本轮必须出现的主要瓶颈。当前聚合没有共享锁，不要求 mutex 图出现业务热点。

画像优先选择正式窗口中段，避开边界采样混入数据准备或清理。query 操作增量除以同期成功完成的权限请求数应约为 3；分母也可使用同窗请求指标。若请求失败，或连接等待上升，结合错误率与吞吐解释计数。5000 或 25000 行权限的数据库扫描仍可能占据明显开销，业务聚合不是接口的全部耗时。

图表与画像来源见[观测数据速查](observability-data-map.md)。用户在图表中添加 Annotation 时记录本轮算法、角色数、权限数、RPS 与正式窗口。

### 中断与清理

客户端运行期间 Ctrl+C 会取消 Vegeta，并尝试自动清理。客户端被强制终止或清理失败时，按日志中的清单路径恢复：

```shell
./bin/pgo performance permissions cleanup .local/performance/permissions/<时间戳>/01-permissions.json
```

清理命令重新获取批次用户令牌，已清理清单重复执行不会新建数据。准备阶段响应丢失或中断时，通过既有列表接口找回批次数据，只删除该批次的记录；权限列表接口当前仅支持 ID 筛选，恢复阶段需要读取权限列表后按本批次角色过滤，正常完成的清理只使用记录好的 ID。没有清单时不能自动定位批次，保留运行目录直到确认清理成功。

### 本地业务算法验证

```shell
go test ./internal/userService/service -run '^$' -bench '^BenchmarkPermissionAggregation$' -benchmem
```

基准只测业务聚合，帮助确认缺陷可观测，不能代替真实 HTTP 压测或平台图表验收。

## 待完善

- [ ] 单元测试覆盖核心业务逻辑
- [ ] 端到端测试覆盖关键用户流程
- [ ] CI 中集成自动化测试
- [ ] 发布前检查清单

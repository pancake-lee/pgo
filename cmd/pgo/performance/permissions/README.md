# 共享角色权限 HTTP 读写压测

设计与业务规模见[独立设计](../../../../docs/design/2026-10-03-01-permission-read-write-performance.md)，任务交接见[backlog 47](../../../../docs/backlog.md#47-共享角色权限读写混合实验)。

## 运行前

使用隔离测试环境，用户自行启动更新后的 userService。默认权限读取使用 JOIN；后续比较分表查询时，将[权限服务](../../../../internal/userService/service/permission.go)中的内部常量 `usePermissionJoin` 改为 `false`，重新编译并重启服务。恢复 JOIN 时改为 `true`，同样重新编译并重启。

两种查询实现都使用 gentool 生成的表、字段和查询接口，只读取动作与路径，重复动作按权限主键顺序覆盖。当前不会创建或删除索引，首次保留数据库真实状态。

开启额外采样时，诊断端口需开启 `Diagnostics.Enabled`、`Diagnostics.Pprof`，并为 `BlockProfileRate`、`MutexProfileFraction` 配置正数，模板均为 1；端口保持受控可达。portal 中的 diagnostics 地址用于控制测试采样，Alloy 的配置需同步到部署环境。

负载工具沿用现有 Vegeta v12.13.0。服务启动方式沿用现有部署；前台进程通过 Ctrl+C 停止，受控后台进程记录 PID 并使用 `kill <PID>` 停止。

## 首次运行

菜单中的 Performance 分组提供“登录性能测试”“登录数据清理”“权限性能测试”“权限数据清理”四项。同一场景的测试和清理共享 `output directory` 缓存；使用同一目录即可复用数据。

先选定目录，通过 HTTP 接口准备固定数据：

```sh
PERMISSION_OUTPUT=.local/performance/permissions/comparison
./bin/pgo performance permissions http://127.0.0.1:20080 \
  --output-dir "$PERMISSION_OUTPUT" --prepare-only
```

将地址替换为实际 portal 地址。固定数据为 20 个项目、每项目 20 个角色、每角色配置相同的 100 个权限点，以及 1000 名普通用户和 1 名管理员。每项目两个读组各 25 人、各共享 5 个角色。每次查询读取 500 条关联记录，合并为 100 项权限。

权限创建采用 8 个并发 HTTP 请求；其他创建按关系依次执行。控制台输出准备开始、每项目完成进度、令牌刷新、权限校验、目标生成与预热提示。准备统计只保留创建数量、总耗时、平均墙钟耗时及验证成功汇总。平均墙钟耗时是总耗时除以成功记录数，接口延迟仍以监控为准。

后续指定同一输出目录，程序自动识别清单，避免每次重复创建 40000 条权限关联记录：

```sh
./bin/pgo performance permissions http://127.0.0.1:20080 \
  --output-dir "$PERMISSION_OUTPUT" --rps 20 --duration 120s --write-rps 1
```

每次运行时，两个读组与管理员修改同时预热超过 30 秒，延续至目标整分钟前 5 秒，停止预热并等待在途请求和当前角色轮次收尾，空等到 00 秒再同时测量指定时长。若收尾错过整分钟，日志记录延迟与实际开始时间。菜单说明读写同时运行并读取管理员写入 RPS（默认 1，记住上次值），随后读取公共负载参数。`--rps` 是两个读组的总速率，20 表示每组 10 RPS；`--write-rps` 是单条管理员修改请求速率。管理员只修改热点角色，停止写入时完成当前动作的角色轮次，再验证最终值。

每次运行执行一次读写负载。自动升压使用 `--rps auto`，按 200、400、600、800、1000 的固定阶梯运行，每档执行一次读写负载并分别保存报告；整轮只准备或加载一次数据，不生成跨档汇总。需要更低起点时仍可指定数值 RPS。

```sh
./bin/pgo performance permissions http://127.0.0.1:20080 \
  --output-dir "$PERMISSION_OUTPUT" --rps auto --duration 120s
```

同目录数据满足当前规模时会检查实际记录、刷新令牌并直接复用；规模变化、准备未完成、清理未完成或数据缺失时，先通过 HTTP 删除旧批次，再重建。清理失败保留原清单并停止，下一次运行可继续；请求或鉴权错误直接报错，不当作数据不足。默认保留测试数据，也可使用 `--keep-data=false` 在结束后清理。

CLI 省略 `--output-dir` 时仍生成新目录；要自动复用需明确指定同一目录，菜单会记住目录。保留 `--manifest` 供已有调用指定清单，该方式独立清理应使用原清单路径。

## 看哪些结果

客户端只保存负载侧结果，包含运行参数、HTTP 目标、两个读组的 Vegeta 结果与报告，以及管理员请求报告。清单和目标文件含测试令牌，文件权限为 0600。

准备清单与两组 HTTP 目标固定保存到输出目录的 `data/`。正式测试开始准备数据前，只清除根目录上次的 `hot/`、`control/`、`writer-report.txt` 及自动档位报告，保留 `data/`、`round-xx/` 和用户截图。`00-run.json` 更新为本次运行参数。新测试中途失败时，未执行窗口不会显示上次结果。不同 RPS 使用同一目录即可复用同一套准备数据。`--prepare-only` 保留已有负载报告。

auto 下每档结果放在 `rps-200/`、`rps-400/` 等子目录中，结构如下；清单及 HTTP 目标始终放在输出目录的 `data/` 并由各档复用。`round-xx/` 由用户确认结果后手动迁移，程序不创建、不清理；迁移结果时保留 `data/`。

```text
本次输出目录/
  data/
    01-permissions.json
    hot-targets.jsonl
    control-targets.jsonl
  00-run.json
  hot/11-vegeta-report.txt
  control/11-vegeta-report.txt
  writer-report.txt
  round-01/                 # 手动备份结果，不包含 data/
```

公共参数 `--sampling` 默认关闭，菜单中的 `sampling` 默认 `false` 并记住上次值；CLI 用 `--sampling` 开启、`--sampling=false` 关闭。先用普通负载寻找异常，再保持 RPS、数据和测试时长相同，开启采样复测。开关控制 block/mutex 事件采样，CPU/heap 与 goroutine 的 Alloy 抓取沿用配置。

开启时，测试在预热与分钟对齐完成后、正式负载开始前开启后端 block/mutex 采样，负载收尾或取消时关闭。采样时长取正式测试时长，由后端截断到 24 小时上限；超长测试继续执行，采样只覆盖正式负载前段。日志显示后端返回的实际采样时长，正式窗口按控制请求完成后的实际启动时间记录。开启失败会终止测试，关闭失败会报错，会话到期会自动关闭；`--prepare-only` 不开启采样。Alloy 持续抓取，不在客户端下载或上传 profile。

控制台打印正式窗口的 UTC 起止时间。按这些时间在 Grafana 中看权限读取和权限修改的请求 P95、错误率、数据库操作耗时、连接池等待、进程 CPU 和分配；在 Pyroscope 看同实例的 CPU、分配、goroutine、block 与 mutex 画像。goroutine 为快照，block/mutex 为约 14 秒增量；采样关闭后仍正常抓取，没有新事件时不会重放累计历史。准备阶段可以直接观察创建接口的 P95，客户端不额外建立准备阶段报告。

首次正常成功时，JOIN 每个读请求约一次查询，batch 完整路径约三次查询；管理员每次修改有一次更新和一次回读。查询操作总量应扣除管理员回读贡献，连接池等待不能直接解释为 MySQL 行锁等待。

首次结果若没有明显瓶颈，记录该结果后再升压；索引优化和 JOIN/batch 比较按单个因素逐轮进行。数据库内部执行计划需要数据库侧查看，Go CPU profile 不包含 MySQL 内部 JOIN 开销。

## 清理

```sh
./bin/pgo performance permissions-cleanup "$PERMISSION_OUTPUT"
# 同样可用场景子命令
./bin/pgo performance permissions cleanup "$PERMISSION_OUTPUT"
```

清理只需目录，从清单读取服务地址，不要求 portal 地址、RPS 或 Vegeta。没有清单或已完成清理时直接成功。清理通过现有 HTTP 接口删除本批次记录，不修改表结构。中断后保留清单，清理时可按批次名称、项目及角色关系找回已入库但尚未记入最后一批清单的记录；这会读取现有列表接口的数据，发生在正式测量窗口之外。

创建请求前先保存批次身份；首次管理员登录响应丢失、准备中断或最后删除后清单尚未更新时，清理入口可以重新登录同批次管理员并恢复剩余记录。损坏且无法识别批次的清单直接报错，避免覆盖恢复线索。正式读报告有非成功响应或管理员修改失败时结束本档，保留结果用于排查。

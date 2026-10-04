# 共享角色权限 HTTP 读写压测

设计与业务规模见[独立设计](../../../../docs/design/2026-10-03-01-permission-read-write-performance.md)，任务交接见[backlog 47](../../../../docs/backlog.md#47-共享角色权限读写混合实验)。

## 运行前

使用隔离测试环境，用户自行启动更新后的 userService。默认权限读取使用 JOIN；后续比较分表查询时，将[权限服务](../../../../internal/userService/service/permission.go)中的内部常量 `usePermissionJoin` 改为 `false`，重新编译并重启服务。恢复 JOIN 时改为 `true`，同样重新编译并重启。

两种模式都使用 gentool 生成的表、字段和查询接口，只读取动作与路径，重复动作按权限主键顺序覆盖。当前不会创建或删除索引，首次保留数据库真实状态。

负载工具沿用现有 Vegeta v12.13.0。服务启动方式沿用现有部署；前台进程通过 Ctrl+C 停止，受控后台进程记录 PID 并使用 `kill <PID>` 停止。

## 首次运行

先通过 HTTP 接口准备固定数据：

```sh
./bin/pgo performance permissions http://127.0.0.1:20080 --prepare-only
```

将地址替换为实际 portal 地址。固定数据为 20 个项目、每项目 20 个角色、每角色配置相同的 100 个权限点，以及 1000 名普通用户和 1 名管理员。每项目两个读组各 25 人、各共享 5 个角色。每次查询读取 500 条关联记录，合并为 100 项权限。

权限创建采用 8 个并发 HTTP 请求；其他创建按关系依次执行。控制台输出准备开始、每项目完成进度、令牌刷新、权限校验、目标生成与预热提示。准备统计只保留创建数量、总耗时、平均墙钟耗时及验证成功汇总。平均墙钟耗时是总耗时除以成功记录数，接口延迟仍以监控为准。

程序打印清单路径，后续使用同一清单，避免每次重复创建 40000 条权限关联记录：

```sh
PERMISSION_MANIFEST=.local/performance/permissions/<本次目录>/01-permissions.json
./bin/pgo performance permissions http://127.0.0.1:20080 \
  --manifest "$PERMISSION_MANIFEST" --rps 20 --duration 120s --write-rps 1
```

每次运行依次执行纯读、混合、恢复三个窗口，每窗口预热 30 秒，再测量指定时长。`--rps` 是两个读组的总速率，20 表示每组 10 RPS；`--write-rps` 是混合窗口的单条管理员修改请求速率。管理员只修改热点角色，停止写入时完成当前动作的角色轮次，再验证最终值。

首次运行一轮即可检查数据与整个流程。重复对照可添加 `--repeat 3`，同一批次执行三轮。自动升压使用 `--rps auto`，按 200、400、600、800、1000 的固定阶梯运行，每档执行三个窗口并分别保存报告；整轮只准备或加载一次数据，不生成跨档汇总。需要更低起点时仍可指定数值 RPS。

```sh
./bin/pgo performance permissions http://127.0.0.1:20080 \
  --manifest "$PERMISSION_MANIFEST" --rps auto --duration 120s
```

如果省略 `--prepare-only` 和 `--manifest`，会准备新批次并直接运行。默认保留测试数据用于后续对照；也可使用 `--keep-data=false` 在结束后清理。

## 看哪些结果

客户端只保存负载侧结果，包含运行参数、HTTP 目标、两个读组的 Vegeta 结果与报告，以及混合窗口的管理员请求报告。清单和目标文件含测试令牌，文件权限为 0600。

auto 下每档结果放在 `rps-200/`、`rps-400/` 等子目录中，结构如下；清单及 HTTP 目标放在整轮根目录并由各档复用。

```text
本次输出目录/
  round-01/
    pure/hot/11-vegeta-report.txt
    pure/control/11-vegeta-report.txt
    mixed/hot/11-vegeta-report.txt
    mixed/control/11-vegeta-report.txt
    mixed/writer-report.txt
    recovery/hot/11-vegeta-report.txt
    recovery/control/11-vegeta-report.txt
```

控制台打印正式窗口的 UTC 起止时间。按这些时间在 Grafana 中看权限读取和权限修改的请求 P95、错误率、数据库操作耗时、连接池等待、进程 CPU 和分配；在 Pyroscope 看同实例的 CPU 与分配画像。准备阶段可以直接观察创建接口的 P95，客户端不额外建立准备阶段报告。

首次正常成功时，JOIN 每个读请求约一次查询，batch 完整路径约三次查询；管理员每次修改有一次更新和一次回读。混合窗口查询数应扣除管理员回读贡献，连接池等待不能直接解释为 MySQL 行锁等待。

首次结果若没有明显瓶颈，记录该结果后再升压；索引优化和 JOIN/batch 比较按单个因素逐轮进行。数据库内部执行计划需要数据库侧查看，Go CPU profile 不包含 MySQL 内部 JOIN 开销。

## 清理

```sh
./bin/pgo performance permissions cleanup "$PERMISSION_MANIFEST"
```

清理通过现有 HTTP 接口删除本批次记录，不修改表结构。中断后保留清单，清理时可按批次名称、项目及角色关系找回已入库但尚未记入最后一批清单的记录；这会读取现有列表接口的数据，发生在正式测量窗口之外。

首次用户数据准备失败时，程序已打印清单路径；如果管理员登录也没有成功完成，则需先核对该批次管理员记录，不能直接使用完整清理流程。正式读报告有非成功响应或管理员修改失败时结束本档，保留结果用于排查。

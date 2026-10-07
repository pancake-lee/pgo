# 权限压测瓶颈定位记录

> 专题入口：[性能测试闭环中枢](../../../design/2026-09-30-01-login-performance-hub.md)。本记录汇总 round-01～round-02，索引优化效果待下一轮验证。

## 两轮测试

- **round-01，建立可分析的成功窗口**：北京时间 2026-10-07 17:02–17:03，读 40 RPS、写约 5 RPS。两组各 1200 次读取全部成功，[热点组](round-01/hot/11-vegeta-report.txt)与[对照组](round-01/control/11-vegeta-report.txt)整体 P95 分别约 155ms、166ms。末尾短窗口 SQL P95 升到约 568ms，协程从约 36 升到 71，结束后回落。Pyroscope 显示增长主要来自 HTTP 连接；MySQL startWatcher 每次快照保持 2 个，其大额 block delay 是监听完成/取消的累计等待，未显示它是阻塞业务的原因。
- **round-02，提高负载复现错误**：北京时间 2026-10-07 20:29–20:30，读 50 RPS、写约 5 RPS。两组各执行 1500 次读取：[热点组](round-02/hot/11-vegeta-report.txt)有 111 次 HTTP 500，成功率 92.60%；[对照组](round-02/control/11-vegeta-report.txt)有 105 次 HTTP 500，成功率 93%。两组 P95 均约 1.002 秒，失败比例接近；[写入报告](round-02/writer-report.txt)中 300 次请求全部成功。SQL 短窗口 P95 约 1.60 秒，MySQL CPU 约 1.85 个核心，主机 CPU 约 84%，应用 CPU 约 0.20 个核心；连接使用数一度达到 43，协程达到 176，结束后回落。Pyroscope 出现 GetUserPermissions → MySQL readPacket 的等待栈，将调查重点收敛到数据库返回路径。

round-01 两组延迟接近，round-02 两组又出现接近的失败比例，更支持共同查询路径的资源压力，尚未显示热点角色写入对热点组造成明显额外影响，不能据此认定锁竞争。两轮均使用约 5 RPS 写入、60 秒正式负载并开启采样，读负载由 40 RPS 提高到 50 RPS。第二轮约 1 秒返回 500，更符合后端约 1 秒 deadline。

## 执行计划将范围缩小到缺失索引

[round-02/explain.txt](round-02/explain.txt)中的同一条 JOIN 显示：

- user_role_assoc 使用 idx_user_role，估计读取 5 条角色关联。
- user_role 使用 PRIMARY，按主键关联并筛选项目。
- user_role_permission_assoc 的 type=ALL、possible_keys=NULL、key=NULL，估计扫描 38473 行，再执行 hash join 和过滤；缺少可用于 role_id 关联的索引。对 InnoDB 表而言，这种全表扫描沿聚簇索引读取表中行，并不等于每次都发生物理磁盘读取。

每个用户目标约为 5 个角色对应的 500 条权限关联，却需要扫描约 4 万条记录。按读 50 RPS 粗略估算，重复扫描规模约为每秒 200 万行，尚未计入管理员回读；EXPLAIN 的 rows 是估计值，实际行数与循环次数待 EXPLAIN ANALYZE 验证。

同一 SQL 独立执行约 27ms，是用户在无压力时的实测。单次耗时较短仍可能包含大量无效扫描；并发下重复扫描争抢 CPU 等资源，查询变慢、在途连接与协程增加，最后超过请求 deadline。27ms 是墙钟时间，不能直接当作 CPU 消耗。Using temporary/Using filesort 是附加成本，现阶段不将它们判定为主因或磁盘排序。

**当前推断**：首要瓶颈是 user_role_permission_assoc 缺少 role_id 索引，导致权限读取重复全表扫描，并在并发下放大查询成本。执行计划、MySQL CPU 和数据库读取等待相互支持这一解释；缺索引的实际影响仍需索引对照确认，尚未证明锁竞争或协程泄漏。

## 下一轮索引对照

用户建立 role_id 索引后，保持 JOIN、同一套 data/、读 50 RPS、写 5 RPS、正式时长 60 秒、timeout 与采样配置不变。新结果保存在 current/，确认后手动归档至下一组 round-xx。

先在负载窗口外记录新执行计划，确认是否改用 role_id 索引及实际扫描行数，再对比两组读成功率/P95、写入延迟、MySQL CPU 和连接/协程峰值。若扫描量和 CPU 明显下降且同负载不再超时，便能增强缺索引是本轮主要瓶颈的因果证据。

# 嵌套事务设计

> 状态：已实现

## 目标

为当前服务的跨层写操作提供可嵌套事务。业务代码自行选择并传递 `transactionID`；数据访问层通过该 ID 获取当前事务的 GORM Gen 查询对象，无需传递 `*gorm.DB` 或改造既有 DAO。

## API

```go
transactionID := requestID // 由业务保证与其他并发流程不冲突
tx, err := pdb.Begin(transactionID)
if err != nil {
    return err
}
defer tx.Rollback()

query := db.GetQueryTx(transactionID)
// ... query.User.Create / Updates ...
return tx.Commit()
```

- `Begin(transactionID)` 首次调用开启真实 GORM 事务；之后的同 ID 调用只创建逻辑 handle。
- `pdb.GetGormDB(transactionID)` 和 `db.GetQueryTx(transactionID)` 在事务活跃时返回根事务对象；不存在事务时返回默认连接。
- 原有 `pdb.GetGormDB()` 与 `db.GetQuery()` 保持不变，因此存量 DAO 不需要迁移。
- `-1` 是保留的默认 ID，不能用于开启事务。

## 收口规则

每个 handle 独立记录 `pending`、`committed` 或 `rolled back` 状态，而非依赖调用顺序：

```text
Begin(42) -> root handle，开启真实事务
Begin(42) -> inner handle，复用真实事务
root.Commit()  -> root 标记 committed，事务保持打开
inner.Commit() -> 所有 handle 已完成，真实事务 commit

任意 handle.Rollback() -> 标记 rolled back
全部 handle 完成      -> 真实事务 rollback
```

因此外层与内层无需按 LIFO 顺序关闭。handle 的重复 `Commit` / `Rollback`，以及显式 `Commit` 后的 `defer Rollback` 都会安全忽略。

## 并发

注册表、handle 列表和各 handle 状态由同一把互斥锁保护。`Begin`、查询对象查找、状态变更和最终删除对其他 goroutine 原子可见，不会发生 map、slice 或状态字段的并发读写。

同一个 ID 应只用于同一业务工作流；不同并发工作流必须使用不同 ID。

## 渐进接入

新增的事务链路在需要写数据库的位置使用 `db.GetQueryTx(transactionID)`；其余 DAO 继续使用 `db.GetQuery()`。后续若希望自动化，可在业务 `AppCtx` 增加 `TransactionID` 字段，并让代码生成模板输出 `GetQueryTx(ctx.TransactionID)`，但这不是本次接入的前置条件。

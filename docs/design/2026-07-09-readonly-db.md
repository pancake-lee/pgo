# 数据库只读连接统一封装

> 状态：设计完成，待实现

## 背景

部分业务场景需要对外暴露 SQL 执行能力（如 Text-to-SQL、BI 查询面板），需要从数据库引擎层面保证只读，避免：

- 字符串层面的 SQL 审计绕过风险（注释注入、UNION SELECT、CTE 绕过等）
- AI 生成 SQL 的意外写入或破坏

**核心思路**：不在 Go 代码层做 SQL 关键字审计，而是利用各数据库引擎自身的只读机制，在内核层面物理拒绝写操作。

## 各驱动只读机制分析

### SQLite

`glebarez/go-sqlite`（v1.21.2）支持 SQLite 标准的 URI 参数 `mode=ro`，驱动测试代码中有直接验证：

```go
// glebarez/go-sqlite@v1.21.2/all_test.go
rodb, err := sql.Open(driverName, fmt.Sprintf("file:%s?mode=ro", name))
```

这是 SQLite VFS 层面的限制，连接打开后任何写操作都会被 SQLite 内核拒绝，返回 `SQLITE_READONLY` 错误。

### MySQL

`go-sql-driver/mysql` 没有原生的只读 DSN 参数。选用方案：连接初始化时执行 `SET SESSION TRANSACTION READ ONLY`。

- MySQL 8.0+ 不需要特殊权限
- 作用域为该连接的整个生命周期，后续所有事务均为只读
- 实现方式：通过 `database/sql` 的 `Connector` 接口或 GORM 的 `Plugin` 钩子在连接建立后执行初始化 SQL

其他考虑过但排除的方案：

- **只读 MySQL 用户**：需要 DBA 运维配合，不属于代码层面控制
- **`SET GLOBAL read_only = 1`**：影响整个实例，太粗暴

### PostgreSQL

pgx 驱动原生支持 DSN 参数 `default_transaction_read_only=on`：

```
host=xxx user=xxx dbname=xxx default_transaction_read_only=on
```

连接建立后自动进入只读事务模式，无需额外初始化 SQL。

## 统一 API 设计

### 新增接口

```go
// pkg/pdb/pdb.go

// DriverType 数据库驱动类型，Init* 时设置
type DriverType string

const (
    DriverSQLite    DriverType = "sqlite"
    DriverMySQL     DriverType = "mysql"
    DriverPostgres  DriverType = "postgres"
)

// GetGormDB_RO 返回只读 *gorm.DB，首次调用时自动懒初始化。
// 初始化失败返回 nil，后续调用会重试。
func GetGormDB_RO() *gorm.DB

// GetDB_RO 返回只读 *sql.DB，首次调用时自动懒初始化。
// 初始化失败返回 error，后续调用会重试。
func GetDB_RO() (*sql.DB, error)
```

### 懒初始化策略

与 `gDB`（服务启动时主动初始化）不同，RO 连接采用**懒初始化**：

- `GetGormDB_RO` / `GetDB_RO` 首次调用时，判断内部 `gReadOnlyDB` 是否为 nil
- 若未初始化，使用 `Init*` 时保存的连接参数（DSN / 文件路径），附加只读参数创建新连接
- 使用 double-checked locking（`sync.Mutex`）保证并发安全
- 初始化失败不阻塞后续调用，下次 Get 会重新尝试

### 内部实现逻辑

```
GetGormDB_RO() / GetDB_RO()
  │
  ├─ gReadOnlyDB != nil → 直接返回
  │
  └─ gReadOnlyDB == nil → 加锁，再次判断，调用 initReadOnlyDB()
        │
        ├─ driverType == DriverSQLite
        │     gorm.Open(glebarez.Open("file:" + savedPath + "?mode=ro"), ...)
        │
        ├─ driverType == DriverMySQL
        │     mysql.NewConnector(cfg) → 包裹 roMySQLConnector
        │     sql.OpenDB(roConnector)
        │     → roMySQLConnector.Connect() 中每次新连接执行:
        │         SET SESSION TRANSACTION READ ONLY
        │     → gorm.Open(mysql.New(mysql.Config{Conn: roSqlDB}), ...)
        │
        └─ driverType == DriverPostgres
              gorm.Open(postgres.Open(savedDSN + " default_transaction_read_only=on"), ...)
```

### 单例管理

| 变量 | 类型 | 用途 | 初始化时机 |
| ---- | ---- | ---- | ---------- |
| `gDB` | `*gorm.DB` | 读写连接 | 服务启动时主动 `Init*` |
| `gReadOnlyDB` | `*gorm.DB` | 只读连接 | 首次 `GetGormDB_RO` / `GetDB_RO` 时懒初始化 |
| `gDriverType` | `DriverType` | 驱动类型标记 | `Init*` 时设置 |
| `gSavedDSN` | `string` | MySQL/PG 的连接串 | `Init*` 时保存 |
| `gSavedFilePath` | `string` | SQLite 文件路径 | `Init*` 时保存 |

两个连接各自独立，互不影响。RO 连接复用 `Init*` 时保存的参数，通过驱动层面的只读机制确保安全。

### Init* 需要保存的信息

当前 `InitSqlite`/`InitMysql`/`InitPG` 各自保存到 `gConf *SqlConfig`，但信息不够完整（没有保存 DSN 或文件路径），需要在 `Init*` 中额外保存：

| 驱动       | 需要保存                | 用途                                           |
| ---------- | ----------------------- | ---------------------------------------------- |
| SQLite     | 原始文件路径`absPath` | 拼接`file:path?mode=ro`                      |
| MySQL      | 原始 DSN 字符串         | `InitReadOnlyDB` 用相同 DSN 再开一条连接     |
| PostgreSQL | 原始 DSN 字符串         | 在末尾追加`default_transaction_read_only=on` |

建议新增内部变量：

```go
var gDriverType    DriverType  // Init* 时设置
var gSavedDSN      string       // MySQL / PostgreSQL 保存完整 DSN
var gSavedFilePath string       // SQLite 保存文件绝对路径
```

### 使用示例

```go
// main.go 或初始化代码
pdb.InitMysqlByConfig()     // 主连接（读写），启动时执行

// 业务代码 —— 不需要显式调用 InitReadOnlyDB，首次 Get 自动完成
func (s *QueryServer) ExecuteSQL(ctx context.Context, req *api.ExecuteSQLRequest) (*api.ExecuteSQLResponse, error) {
    roDB, err := pdb.GetDB_RO()
    if err != nil {
        return nil, err
    }
    rows, err := roDB.QueryContext(ctx, req.Sql)
    // ...
}
```

## 与 GORM 的关系

本次只封装 `*sql.DB` 的只读连接，不封装 `*gorm.DB`。

理由：`ExecuteSQL` 场景执行的是用户/AI 传入的原始 SQL，不需要 GORM 的 ORM 能力，直接用 `database/sql` 即可。如果后续有只读 ORM 操作需求，可以在 `*sql.DB` 基础上再包一层 `gorm.Open`。

## 对 Init* 的改动范围

三组 `Init*` 函数需做统一调整（改动极小，不影响现有逻辑）：

1. **设置 `gDriverType`**：`InitSqlite` 设为 `DriverSQLite`，`InitMysqlByDsn` 设为 `DriverMySQL`，`InitPG` 设为 `DriverPostgres`
2. **保存连接参数**：`InitMysqlByDsn` 保存 DSN 字符串，`InitPG` 保存 DSN 字符串，`InitSqlite` 保存文件绝对路径
3. **不影响现有逻辑**：`gDB` 和 `gConf` 的赋值逻辑不变，向后兼容

## 后续考虑

- `canal_client.go`（MySQL binlog 同步）与只读连接无关，不受影响
- `dbexport` 工具需要读写连接，不使用此接口
- 如果需要在连接池层面做更多控制（如最大连接数、超时），可在 `InitReadOnlyDB` 中通过 `*sql.DB` 的标准方法配置

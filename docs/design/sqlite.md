# SQLite 兼容设计

## 概述

`genGORM` 和 `genCURD` 工具支持使用 SQLite 作为表结构定义源，生成 GORM Model/Query 代码和 CURD 代码。底层 GORM 和 `gorm.io/gen` 已天然支持 SQLite，本项目在此之上增加了连接层的适配和参数路由。

## 数据库类型参数

`genGORM` 和 `genCURD` 均支持 `-db` 参数（默认 `mysql`）：

- `-db mysql`：DSN 为 MySQL 连接字符串（`user:pass@tcp(host:port)/dbname`）
- `-db sqlite3`：DSN 为 SQLite 文件路径（`/path/to/db.sqlite` 或相对路径 `./testdata/mydb.sqlite`）

## 架构与兼容范围

### 已兼容

| 模块 | 说明 |
|---|---|
| `pkg/pdb/sqlite.go` | SQLite 连接初始化，用 `gorm.io/driver/sqlite` 打开文件 |
| `genGORM` | 读取 SQLite 表结构生成 GORM Model + Query 代码 |
| `genCURD` | 读取 SQLite 表结构生成 DAO + Service + Proto + Main 代码 |
| 生成的 CURD 代码 | 通过 GORM 抽象层与具体数据库解耦，天然兼容 SQLite |

### 不兼容（仅 MySQL 支持）

| 模块 | 位置 | 说明 |
|---|---|---|
| **Canal（binlog 同步）** | `pkg/pdb/canal_client.go` | 依赖 MySQL binlog，SQLite 无此机制 |
| **bootCheck / Schema Diff** | `pkg/pdb/mysqlDiff.go`, `pkg/papp/bootCheck.go` | 依赖 `SHOW TABLES`/`SHOW CREATE TABLE` 等 MySQL 专有命令 |
| **dbexport / dbImport** | `pkg/pdb/dbexport/` | 使用 `DESCRIBE` 命令、`mysqldump` 工具、`auto_increment` 检测 |
| **sheet2mysql** | `cmd/pgo/tools/sheet2mysql/` | 生成 MySQL 专有 DDL（`ENGINE=InnoDB`, `AUTO_INCREMENT` 等） |

### 类型映射注意事项

SQLite 使用柔性类型系统（存储类：INTEGER、REAL、TEXT、BLOB、NULL），与 MySQL 的严格类型不同。生成代码时：

- `genGORM`：`gorm.io/gen` 通过 GORM 的 `ColumnTypes()` 获取列类型，SQLite 驱动返回的是建表时的声明类型名。建议 SQLite 建表时使用标准类型名（如 `INTEGER`、`TEXT`、`DATETIME`、`DECIMAL` 等），以便 `genCURD` 的类型映射正常工作。
- `genCURD`：通过 `DatabaseTypeName()` 判断 `date`/`datetime`/`DECIMAL` 来做类型转换。SQLite 驱动下这些名称取决于建表 SQL 中使用的类型名。

## 使用方式

### 直接使用 pgo CLI

```shell
# 生成 GORM 代码
pgo genGORM \
  -db sqlite3 \
  -dsn ./testdata/mydb.sqlite \
  -outPath ./internal/pkg/db/query/ \
  -outFile query.go \
  -modelPkgName model

# 生成 CURD 代码
pgo genCURD -db sqlite3 -dsn ./testdata/mydb.sqlite
```

### 集成到 Makefile

项目默认使用 MySQL，如需切换为 SQLite，修改 Makefile 中 `gorm` 和 `curd` 目标的参数即可：

```makefile
# SQLite database file path (for gorm-sqlite / curd-sqlite targets)
dbPath?=./testdata/pgo.sqlite

# gorm 目标改为：
pgo genGORM \
    -db sqlite3 \
    -dsn "$(dbPath)" \
    -outPath ${dbCodePath}/query/ \
    -outFile query.go \
    -modelPkgName model \

# curd 目标改为：
pgo genCURD -db sqlite3 -dsn "$(dbPath)"
```

MySQL 建库和导入 SQL 文件的步骤（`DROP DATABASE`/`CREATE DATABASE`/`mysql < file.sql`）在 SQLite 场景下不再需要，可移除或注释。

## 实现细节

### `pkg/pdb/sqlite.go`

```go
func InitSqlite(dbPath string) (err error) {
    absPath, err := filepath.Abs(dbPath)
    // ...
    gConf = &SqlConfig{
        Addr:   absPath,
        DbName: filepath.Base(absPath),
    }
    gDB, err = gorm.Open(sqlite.Open(absPath), &gorm.Config{
        SkipDefaultTransaction: true,
    })
    // ...
}
```

参照 `mysql.go`/`pgsql.go` 的模式，初始化全局 `gDB` 和 `gConf`。SQLite 无需 host/user/password，DSN 就是文件路径。

### 参数路由

`genGORM.Run()` 和 `genCURD.runGenerate()` 根据 `-db` 参数 dispatch：

```go
switch dbType {
case "mysql":
    err = pdb.InitMysqlByDsn(dsn)
case "sqlite3":
    err = pdb.InitSqlite(dsn)
default:
    return fmt.Errorf("db %q is not supported", dbType)
}
```

## 后续规划

- `sheet2mysql`：增加 `BuildSqliteCreateTableSQL` 变体，适配 SQLite DDL 语法差异（`AUTOINCREMENT`、无 ENGINE/CHARSET 等）

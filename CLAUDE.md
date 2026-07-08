# CLAUDE.md

## 编译与测试

```shell
# 构建全部
make build

# 仅构建 CLI（当前平台）
make cli

# 跨平台构建 Windows
make cli-win

# 生成 ORM 代码
make gorm

# 生成 CURD 代码
make curd

# 生成 API proto 代码
make api

# 生成客户端 SDK
make api-cli

# 初始化数据库
make initDB

# 提交生成代码
make precommit
```

- 所有可执行程序输出到 `./bin/`，不要直接 `go build` 到根目录
- 本地环境一般设置好了 `GOTOOLCHAIN=local`，不要修改该配置，不使用 Go 自动工具链下载

## 代码风格

### 命名规范

- **列表/切片**：后缀 `List`（如 `userRoleList`、`permissionList`）
- **Map 结构**：后缀 `Map`（如 `roleIDMap`、`permissionMap`），更清晰时用 `keyToValueMap`（如 `idToUserMap`）
- **函数命名**：统一用"动宾"结构
  - C: `add` — 新增，尽量让一种数据的创建入口尽可能少
  - U: `edit` — 主动修改；`update` — 被动更新
  - R: `get` — 查询
  - D: `del` — 删除
  - 关联关系：`addXxxToYyy` / `delXxxFromYyy`
- **HTTP method**：GET（查询）、POST（创建）、PUT（全量更新）、PATCH（部分更新）、DELETE（删除）

### 格式化

- 逻辑修改后统一用 `gofmt -w <file>` 处理，不纠结缩进对齐

### 代码组织

- 避免 `if` 中使用 `;`（如 `if d, ok := data["k"]; ok`），易造成长代码
- 入口函数放在 `internal/<module>/<module>.go`，而非 `cmd/`
- 非复杂场景优先用基础类型组合，仅在复用明显或封装语义明确时抽象 `type`
- **接口代理模式**：基础类通过接口代理支持子类覆盖，子类初始化后调用 `BindProvider(self)`

## 测试规范

- 编写集成测试验证 Service 层逻辑
- 使用 `defer` + 清理函数移除测试数据
- **禁止**在测试中修改表结构，差异应正常报错以提醒升级注意

## 禁忌清单

- 不要在测试中修改表结构
- 不要直接 `go build` 到根目录
- `bootCheck` 的 MySQL 检查不允许执行 `DROP`、`TRUNCATE`、`ALTER ... MODIFY/CHANGE/RENAME`、删索引、删主键等危险 SQL
- 数据库结构不能 drop，仅代码层面废弃，实际数据清理由用户自行操作

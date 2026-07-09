# Sponge 代码生成工具评估

> 状态：已拒绝

## 背景

刚开始做这个库时，代码生成流程是：新增表 → gorm-gen 生成数据库代码 → 自己写工具生成 protobuf 结构体和基础 CURD 接口 → protoc 生成接口代码 → kratos 提供服务。

后来了解到 [Sponge](https://github.com/zhufuyi/Sponge/)，发现它基本就是这个思路的现成实现，于是进行了试用评估。

## 试用过程

使用的 Sponge 版本是 v1.10.3，通过以下方式生成代码：

```shell
Sponge web http \
  --module-name=gogogo \
  --server-name=service_user \
  --project-name=gogogo \
  --db-driver=postgresql \
  --db-dsn=gogogo:gogogo@192.168.101.8:5432/gogogo \
  --db-table=user,user_job,user_dept,user_dept_assoc \
  --suited-mono-repo \
  --extended-api
```

遇到的问题：
- 表字段 COMMENT 中包含换行会导致生成的代码编译失败
- 关联表要求必须有 id 自增列
- embed 参数无法关闭，生成的 dao test 代码总带有 CreatedAt 等列
- 端口修改涉及多个文件，make docs 会还原端口为 8080

## 拒绝原因

核心问题是**生成的代码完全是固定的**，无法适配项目现有技术选型：

1. **框架不匹配**：Sponge 使用 Gin，而本项目使用 Kratos
2. **ORM 使用方式**：生成代码中包含 SQL 字符串（如 `xxx.where("id = ?", id)`），而本项目使用 GormGen 的类型安全方式（`xxx.where(a.ID.Eq(id))`），当列名变更时能通过编译错误快速定位
3. **组件耦合**：业务代码中耦合了框架（Gin）和 ORM（Gorm）的具体类型，替换组件需要改动业务代码

因为这些是"固定生成"无法调整的，所以不适用于实际生产。但 Sponge 的思路仍有参考价值，后续自研 CURD 工具时可借鉴。

## 补充（2024-12-01）

Sponge 更新了自定义模板功能，基于 go template（`{{.xxx}}`）做文本替换。但个人仍偏好模板本身是能编译的 Go 代码，且生成操作需要支持幂等重复执行。未深入试用。

## 参考

- 已归档至 [`docs/archive/rejected.md`](../archive/rejected.md)
- 对应 backlog 中序号 2（genCURD 自研）

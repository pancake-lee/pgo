# 任务管理方案

> 状态：已完成基础后端

## 背景

现有任务管理工具的不足：专业工具对中小型项目/个人使用过于复杂，笔记软件缺乏层级折叠，思维导图在文本格式化方面较弱。

期望一个兼顾树形结构灵活性和表格视图信息密度的轻量工具。

## 产品设计

### 客户端：[Tree World](https://github.com/pancake-lee/tree-world)

- **树视图**：灵活组织父子关系，支持父子任务、备注、备忘
- **表格视图**：展示任务元数据（状态、时间、负责人等）
- **详情展示**：弹窗或抽屉展开表格中难以完整表达的信息

### 后端实现

只需一个任务表，自动生成 CRUD 接口：

1. 定义表结构：编写 SQL，新增 `task` 表
2. `make gorm` 生成数据库访问代码
3. 在 genCURD 代码处添加 `addTable(&model.Task{}, "task")`
4. `make curd` 生成 CURD 接口代码
5. `make build` 编译构建

## 实现要点

- 时间类型用 `timestamp`，扩展 genCURD 支持 `db:timestamp → go:time.Time → pb:int64` 的转换
- HTTP 服务增加 CORS 头
- 增加 logger 模块，封装 zaplog

## 参考

- 对应 backlog 中序号 4

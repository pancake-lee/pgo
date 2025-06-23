# PGO

本仓库是个人习惯的一些封装(pkg)以及个人的小项目。
主要用于学习/练习/沉淀知识，并未打算称为一个“流行框架”

## Feature

### tools/genCURD

这是一个**基于Gorm生成的Model**生成**CURD服务接口代码**的生成工具

- 人工编写，abandon_code对应的sql/proto以及service代码，这是我们的模板。
- abandon命名只是我为了排序第一的灵感单词，刚好也能表达“这不是一个真正的功能模块”。
- abandon_code的代码里包含了类似`// MARK REPLACE XXX START/END`这样的标识
- 根据abandon_code及其`MARK`标识，通过`genCURD`可以生成出其他表的CURD服务接口代码

相比之下，更加常见的是利用`text/template`库，在代码中利用`{{.FieldName}}`表示替换位置。
这种方案中，带有`{{.FieldName}}`的代码无法被VSCode做代码静态分析，也无法被正常编译。
所以除非模板代码已经非常稳定，否则维护这样的模板代码并不容易。

与其他生成工具不同的是，abandon_code相关代码是一份**正常**的代码。
而abandon_code的维护和正常开发几乎没有区别，只需要提供几个`MARK`标识即可。

### userService

该服务实现了一个“组织架构”的功能，包括了用户/部门/职位。

### schoolService

该服务实现了一个“教师课程管理”的功能，主要为了方便教师之间换课。

后端代码仅使用genCURD生成了基础代码，用于保存“换课历史记录”。

而换课等逻辑主要在客户端实现了。所谓的客户端也是go开发的命令行程序而已。`./client/course*.go`

更具体的细节记录在了[docs\backlog\4.md](https://github.com/pancake-lee/pgo/blob/master/docs/backlog/4.md)

### taskService

该服务实现了一个“任务管理”的功能。

这为我另一个项目[tree-world](https://github.com/pancake-lee/tree-world)从当服务端，提供最初的数据源。

## TODO

- 实现 RBAC 权限模型
  - CURD 角色，角色可以继承，并且可以继承多个，但需要做回环检测
  - 角色还有“作用范围”的概念
    - 比如张三有权限打电话给李四，但是在档案室不允许打电话
    - 张三打电话给李四的权限用 RBAC 表达不难：拥有打电话的权限，目标资源包含李四
    - 在档案室打电话的权限，其实可以表达为：拥有打电话的权限，目标资源是档案室
    - 但是理解起来不太顺，我更希望是“用户”，在“指定范围”内，对“指定资源”，拥有“指定权限”

## [commitlint](`https://github.com/conventional-changelog/commitlint`)

| prefix   | desc       |
| -------- | ---------- |
| build    | 构建相关    |
| chore    | 杂项       |
| ci       | CI/CD 相关 |
| docs     | 文档       |
| feat     | 功能       |
| fix      | 修复       |
| perf     | 性能       |
| refactor | 重构       |
| revert   | 回退       |
| style    | 代码风格    |
| test     | 测试       |
| gen      | 生成代码    |
| improve  | 优化代码    |
| tidy     | 整理、清理  |

## [semver](https://semver.org/lang/zh-CN/)

版本格式：主版本号.次版本号.修订号，版本号递增规则如下：

- 主版本号：当你做了不兼容的 API 修改，
- 次版本号：当你做了向下兼容的功能性新增，
- 修订号：当你做了向下兼容的问题修正。
先行版本号及版本编译信息可以加到“主版本号.次版本号.修订号”的后面，作为延伸。

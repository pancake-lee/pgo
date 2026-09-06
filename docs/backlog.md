# Backlog

> 全部技术需求池，按优先级排列。已完成事项见 [v0.0.8](archive/v0.0.8.md)。
> 状态标记：Done、Pending、WIP、Approved、暂缓、Abandoned、Rejected

| 状态 | 序号 | 类别 | 任务 | 简述 |
|------|------|------|------|------|
| Approved | 14 | 代码生成 | genCURD inferServiceName 配置化 | 已确认规则硬编码在生成器中，未提供外部映射配置 |
| Approved | 16 | 代码生成 | genCURD 支持多主键表 | 已确认发现联合主键后清空 PriCol，Update/Delete 不会生成 |

---

## 详细说明

### 14. genCURD inferServiceName 配置化

- **状态**：Approved（已规划）
- **背景**：`inferServiceName` 在 `cmd/pgo/tools/genCURD/genCURD_core.go` 内硬编码 `task`、`course` 等前缀和 `default` 回退；调用方无法覆盖规则。
- **方案**：为 genCURD 增加表名到服务名的显式映射参数，并支持从项目配置读取同一映射；显式参数优先于配置，未命中时保留当前前缀兼容规则并输出提示。开始生成前校验映射格式、重复表名和非法服务名，防止生成到意外目录。
- **任务列表**：
  - 扩展 CLI 参数与配置解析，并定义优先级和兼容回退。
  - 将服务名推断改为可注入规则，保留现有项目的生成结果。
  - 添加映射解析、优先级、非法输入和生成目录的单元测试。
- **验收**：指定表可生成到目标 service；无新配置时已有项目生成结果不变；非法映射在写文件前失败。

### 16. genCURD 支持多主键表

- **状态**：Approved（已规划）
- **背景**：`newTable` 检测到多个主键后将 `PriCol` 置空；DAO、Proto、Service 模板据此移除主键相关的 Update/Delete 逻辑。
- **方案**：将生成器的单一 `PriCol` 模型扩展为有序主键列集合；Update/Delete 请求、服务转换和 DAO 条件都包含全部主键，并只以完整主键组合作为定位条件。保留无主键表的只读生成行为，并以 SQLite 和 MySQL 的联合主键样例回归。
- **任务列表**：
  - 重构表元数据和模板替换逻辑，使其接受主键列集合。
  - 更新 Proto、Service、DAO 生成结果和零值/类型校验。
  - 建立单主键、联合主键、无主键三组 fixture，并验证生成代码可编译及 SQL 条件完整。
- **验收**：联合主键表生成完整 Update/Delete；SQL 条件包含且仅包含全部主键；单主键和无主键行为不回归。
